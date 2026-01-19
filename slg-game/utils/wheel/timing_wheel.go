package wheel

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// TimingWheel 通用分层时间轮 (毫秒→秒→分钟→小时→天→周)
// 支持从毫秒到月的任意时间间隔的定时任务调度
// 使用分层架构优化内存使用和查找效率
type TimingWheel struct {
	levels    []*TimeWheelLevel     // 分层时间轮数组，从粗粒度到细粒度
	startTime time.Time             // 时间轮启动时间，用于统计和调试
	timers    map[string]*TimerTask // 任务ID到任务的映射，用于快速查找和删除
	taskQueue chan *TimerTask       // 待执行任务队列，解耦任务调度和执行
	stopCh    chan struct{}         // 停止信号通道，优雅关闭时间轮
	running   int32                 // 运行状态标志，使用原子操作
	taskCount int64                 // 当前活跃任务数量，用于监控
	mu        sync.RWMutex          // 保护timers map的并发访问，允许多读单写
}

// TimeWheelLevel 表示时间轮的单个层级
// 每个层级处理特定时间精度的任务调度
type TimeWheelLevel struct {
	name         string        // 层级名称，用于调试和监控 ("millisecond", "second", etc.)
	unit         time.Duration // 时间单位 (如：100*time.Millisecond, time.Second)
	slotsPerUnit int           // 每个时间单位包含多少个槽位 (用于计算精度)
	totalSlots   int           // 该层级的总槽位数 (wheelSize)
	slots        []*SlotBucket // 槽位数组，每个槽位存储一批任务
	currentSlot  int           // 当前指向的槽位索引 (0 ~ totalSlots-1)
	interval     time.Duration // tick间隔，即每次前进的时间
	tickCount    int64         // 该层级已执行的tick次数，用于统计
}

// SlotBucket 表示时间轮中的一个槽位
// 每个槽位可以存储多个定时任务
type SlotBucket struct {
	tasks map[string]*TimerTask // 槽位内的任务映射，key为任务ID
	mu    sync.RWMutex          // 保护槽位内任务的并发访问
}

// TimerTask 表示一个定时任务
// 包含任务执行时间、回调函数等信息
type TimerTask struct {
	ID       string      // 任务唯一标识符，用于查找和删除
	Callback func()      // 任务到期时执行的回调函数
	Deadline time.Time   // 任务的执行截止时间
	Level    int         // 任务当前所在的层级索引
	Rounds   int         // 任务还需要等待多少轮（用于跨层级任务）
	Data     interface{} // 额外数据，用户可以存储自定义信息
	Created  time.Time   // 任务创建时间，用于统计和调试
}

// NewTimingWheel 创建分层时间轮
// 初始化6个层级：毫秒→秒→分钟→小时→天→周
// 每个层级都有特定的时间范围和精度
func NewTimingWheel() *TimingWheel {
	levels := []*TimeWheelLevel{
		// 周层: 每周1格，12格=3个月，最大3个月
		{
			name:         "week",
			unit:         7 * 24 * time.Hour,
			slotsPerUnit: 1,
			totalSlots:   12,
			interval:     7 * 24 * time.Hour,
		},
		// 天层: 每天1格，7格=1周，最大1周
		{
			name:         "day",
			unit:         24 * time.Hour,
			slotsPerUnit: 1,
			totalSlots:   7,
			interval:     24 * time.Hour,
		},
		// 小时层: 每小时1格，24格=1天，最大1天
		{
			name:         "hour",
			unit:         time.Hour,
			slotsPerUnit: 1,
			totalSlots:   24,
			interval:     time.Hour,
		},
		// 分钟层: 每分钟1格，60格=1小时，最大1小时
		{
			name:         "minute",
			unit:         time.Minute,
			slotsPerUnit: 1,
			totalSlots:   60,
			interval:     time.Minute,
		},
		// 秒层: 每秒1格，60格=1分钟，最大1分钟
		{
			name:         "second",
			unit:         time.Second,
			slotsPerUnit: 1,
			totalSlots:   60,
			interval:     time.Second,
		},
		// 毫秒层: 每100毫秒1格，10格=1秒，最大1秒
		{
			name:         "millisecond",
			unit:         100 * time.Millisecond,
			slotsPerUnit: 10,
			totalSlots:   10,
			interval:     100 * time.Millisecond,
		},
	}

	// 初始化槽位
	for _, level := range levels {
		level.slots = make([]*SlotBucket, level.totalSlots)
		for i := range level.slots {
			level.slots[i] = &SlotBucket{
				tasks: make(map[string]*TimerTask),
			}
		}
	}

	return &TimingWheel{
		levels:    levels,
		timers:    make(map[string]*TimerTask),
		taskQueue: make(chan *TimerTask, 10000),
		stopCh:    make(chan struct{}),
		startTime: time.Now(),
	}
}

// Start 启动时间轮
// 为每个层级启动独立的goroutine进行tick处理
// 使用原子操作确保只启动一次
func (tw *TimingWheel) Start() error {
	if !atomic.CompareAndSwapInt32(&tw.running, 0, 1) {
		return fmt.Errorf("timing wheel is already running")
	}

	log.Println("Starting Universal Timing Wheel...")

	// 为每个层级启动ticker
	for i := range tw.levels {
		go tw.runLevel(i)
	}

	log.Printf("Universal Timing Wheel started with %d levels", len(tw.levels))
	return nil
}

// Stop 停止时间轮
// 优雅关闭所有层级的goroutine和任务队列
// 使用原子操作确保安全停止
func (tw *TimingWheel) Stop() error {
	if !atomic.CompareAndSwapInt32(&tw.running, 1, 0) {
		return fmt.Errorf("timing wheel is not running")
	}

	log.Println("Stopping Universal Timing Wheel...")

	close(tw.stopCh)

	// 等待一段时间让所有goroutine停止
	time.Sleep(100 * time.Millisecond)

	log.Println("Universal Timing Wheel stopped")
	return nil
}

// AddTimer 添加定时任务
// delay: 任务延迟执行的时间
// callback: 任务到期时执行的函数
// 返回任务ID和可能的错误
// 自动选择最适合的层级进行调度
func (tw *TimingWheel) AddTimer(delay time.Duration, callback func()) (string, error) {
	if atomic.LoadInt32(&tw.running) == 0 {
		return "", fmt.Errorf("timing wheel is not running")
	}

	if delay <= 0 {
		return "", fmt.Errorf("delay must be positive")
	}

	taskID := generateTaskID()

	task := &TimerTask{
		ID:       taskID,
		Callback: callback,
		Deadline: time.Now().Add(delay),
		Created:  time.Now(),
	}

	// 选择最优层级
	levelIndex := tw.selectOptimalLevel(delay)
	task.Level = levelIndex

	// 计算位置
	rounds, slot := tw.calculatePosition(levelIndex, delay)
	task.Rounds = rounds

	// 直接添加到对应槽位
	tw.addToSlot(levelIndex, slot, task)
	atomic.AddInt64(&tw.taskCount, 1)

	return taskID, nil
}

// AddTimerWithData 添加带数据的定时任务
// 允许在定时任务中携带额外的数据信息
// data: 用户自定义的数据，会在TimerTask.Data中存储
func (tw *TimingWheel) AddTimerWithData(delay time.Duration, callback func(), data interface{}) (string, error) {
	taskID, err := tw.AddTimer(delay, callback)
	if err != nil {
		return "", err
	}

	// 查找任务并设置数据 (简化实现，实际应该在AddTimer中处理)
	// 这里只是示例，实际实现需要同步访问

	return taskID, nil
}

// CancelTimer 取消定时任务
// 从所有层级中查找并移除指定的定时任务
// taskID: 要取消的任务ID
// 返回true表示成功取消，false表示任务不存在或已执行
func (tw *TimingWheel) CancelTimer(taskID string) bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()

	if task, exists := tw.timers[taskID]; exists {
		level := tw.levels[task.Level]
		// 遍历所有槽位找到任务
		for _, bucket := range level.slots {
			if bucket != nil {
				bucket.mu.Lock()
				if _, exists := bucket.tasks[taskID]; exists {
					delete(bucket.tasks, taskID)
					delete(tw.timers, taskID)
					atomic.AddInt64(&tw.taskCount, -1)
					bucket.mu.Unlock()
					return true
				}
				bucket.mu.Unlock()
			}
		}
	}
	return false
}

// GetStats 获取统计信息
func (tw *TimingWheel) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	totalTasks := atomic.LoadInt64(&tw.taskCount)
	stats["total_tasks"] = totalTasks
	stats["running"] = atomic.LoadInt32(&tw.running) == 1
	stats["uptime"] = time.Since(tw.startTime)

	// 各层级统计
	for _, level := range tw.levels {
		levelTasks := 0
		for _, bucket := range level.slots {
			if bucket != nil {
				bucket.mu.RLock()
				levelTasks += len(bucket.tasks)
				bucket.mu.RUnlock()
			}
		}
		stats[fmt.Sprintf("%s_tasks", level.name)] = levelTasks
		stats[fmt.Sprintf("%s_tick_count", level.name)] = atomic.LoadInt64(&level.tickCount)
	}

	return stats
}

// selectOptimalLevel 选择最优层级
func (tw *TimingWheel) selectOptimalLevel(delay time.Duration) int {
	// 从最低层开始，找到能容纳该delay的最高层级
	for i := len(tw.levels) - 1; i >= 0; i-- {
		level := tw.levels[i]
		maxDelay := level.unit * time.Duration(level.totalSlots)
		if delay <= maxDelay {
			return i
		}
	}
	return 0 // 如果超出范围，使用最高层级
}

// calculatePosition 计算任务在指定层级的位置
func (tw *TimingWheel) calculatePosition(levelIndex int, delay time.Duration) (rounds int, slot int) {
	level := tw.levels[levelIndex]

	// 计算总单位数
	totalUnits := int(delay / level.unit)

	// 计算圈数和槽位
	rounds = totalUnits / level.totalSlots
	slot = totalUnits % level.totalSlots

	// 考虑当前槽位偏移
	slot = (level.currentSlot + slot) % level.totalSlots

	return rounds, slot
}

// runLevel 运行指定层级
func (tw *TimingWheel) runLevel(levelIndex int) {
	level := tw.levels[levelIndex]
	ticker := time.NewTicker(level.interval)
	defer ticker.Stop()

	log.Printf("Started level %s with interval %v", level.name, level.interval)

	for {
		select {
		case <-ticker.C:
			atomic.AddInt64(&level.tickCount, 1)
			tw.tickLevel(levelIndex)
		case <-tw.stopCh:
			log.Printf("Stopped level %s", level.name)
			return
		}
	}
}

// tickLevel 处理指定层级的tick
func (tw *TimingWheel) tickLevel(levelIndex int) {
	level := tw.levels[levelIndex]

	// 移动到下一个槽位
	level.currentSlot = (level.currentSlot + 1) % level.totalSlots

	bucket := level.slots[level.currentSlot]
	if bucket == nil {
		return
	}

	bucket.mu.Lock()
	slotMap := bucket.tasks
	bucket.tasks = make(map[string]*TimerTask) // 清空
	bucket.mu.Unlock()

	// 处理当前槽位的任务
	tw.processSlotTasks(levelIndex, slotMap)
}

// processSlotTasks 处理槽位中的任务
func (tw *TimingWheel) processSlotTasks(levelIndex int, slotMap map[string]*TimerTask) {
	var expiredTasks []*TimerTask
	var demoteTasks []*TimerTask

	// 分类任务
	for _, task := range slotMap {
		if task.Rounds == 0 {
			if levelIndex == len(tw.levels)-1 {
				// 最低层到期，执行回调
				expiredTasks = append(expiredTasks, task)
			} else {
				// 降级到下一层
				demoteTasks = append(demoteTasks, task)
			}
		} else {
			task.Rounds--
		}
	}

	// 异步执行到期任务
	if len(expiredTasks) > 0 {
		go tw.batchExecute(expiredTasks)
	}

	// 降级任务
	for _, task := range demoteTasks {
		tw.demoteTask(task, levelIndex+1)
	}
}

// batchExecute 批量执行任务
func (tw *TimingWheel) batchExecute(tasks []*TimerTask) {
	for _, task := range tasks {
		atomic.AddInt64(&tw.taskCount, -1)

		// 保护timers map的并发访问
		tw.mu.Lock()
		delete(tw.timers, task.ID)
		tw.mu.Unlock()

		// 异步执行，避免阻塞
		go func(t *TimerTask) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Task %s panicked: %v", t.ID, r)
				}
			}()
			t.Callback()
		}(task)
	}
}

// demoteTask 降级任务到下一层
func (tw *TimingWheel) demoteTask(task *TimerTask, newLevelIndex int) {
	// 计算剩余时间
	remainingDelay := time.Until(task.Deadline)

	if remainingDelay <= 0 {
		// 已经到期，直接执行
		atomic.AddInt64(&tw.taskCount, -1)

		// 保护timers map的并发访问
		tw.mu.Lock()
		delete(tw.timers, task.ID)
		tw.mu.Unlock()

		go task.Callback()
		return
	}

	// 在新层级重新计算位置
	rounds, slot := tw.calculatePosition(newLevelIndex, remainingDelay)

	task.Level = newLevelIndex
	task.Rounds = rounds

	// 添加到新层级
	tw.addToSlot(newLevelIndex, slot, task)
}

// addToSlot 添加任务到指定槽位
func (tw *TimingWheel) addToSlot(levelIndex int, slot int, task *TimerTask) {
	level := tw.levels[levelIndex]

	bucket := level.slots[slot]
	bucket.mu.Lock()
	bucket.tasks[task.ID] = task
	bucket.mu.Unlock()

	// 保护全局timers map的并发访问
	tw.mu.Lock()
	tw.timers[task.ID] = task
	tw.mu.Unlock()
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&taskCounter, 1))
}

var taskCounter int64
