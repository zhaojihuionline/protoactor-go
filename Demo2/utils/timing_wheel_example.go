package utils

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// TimingWheelExample 时间轮使用示例
type TimingWheelExample struct {
	timingWheel *TimingWheel
}

// NewTimingWheelExample 创建示例
func NewTimingWheelExample() *TimingWheelExample {
	return &TimingWheelExample{
		timingWheel: NewTimingWheel(),
	}
}

// Start 启动示例
func (twe *TimingWheelExample) Start() error {
	// 启动时间轮
	if err := twe.timingWheel.Start(); err != nil {
		return err
	}

	log.Println("TimingWheel Example started")

	// 注册一些示例任务
	twe.registerExampleTasks()

	return nil
}

// Stop 停止示例
func (twe *TimingWheelExample) Stop() error {
	return twe.timingWheel.Stop()
}

// registerExampleTasks 注册示例任务
func (twe *TimingWheelExample) registerExampleTasks() {
	// 毫秒级任务 - 技能特效
	twe.scheduleSkillEffect()

	// 秒级任务 - AI决策
	twe.scheduleAIDecisions()

	// 分钟级任务 - 建筑建造
	twe.scheduleBuildingConstruction()

	// 小时级任务 - 玩家状态恢复
	twe.schedulePlayerRecovery()

	// 天级任务 - 日常重置
	twe.scheduleDailyReset()

	// 周级任务 - 服务器维护
	twe.scheduleWeeklyMaintenance()
}

// 毫秒级任务示例 - 技能特效
func (twe *TimingWheelExample) scheduleSkillEffect() {
	duration := 500 * time.Millisecond
	taskID, err := twe.timingWheel.AddTimer(duration, func() {
		fmt.Printf("[%s] 技能特效完成\n", time.Now().Format("15:04:05.000"))
	})

	if err != nil {
		log.Printf("Failed to schedule skill effect: %v", err)
		return
	}

	log.Printf("Scheduled skill effect task: %s, duration: %v", taskID, duration)
}

// 秒级任务示例 - AI决策
func (twe *TimingWheelExample) scheduleAIDecisions() {
	// 安排10个怪物的AI决策
	for i := 0; i < 10; i++ {
		monsterID := i + 1
		// 随机间隔 2-5 秒
		interval := time.Duration(2+rand.Intn(4)) * time.Second

		taskID, err := twe.timingWheel.AddTimer(interval, func() {
			fmt.Printf("[%s] 怪物 %d 进行AI决策\n",
				time.Now().Format("15:04:05"), monsterID)

			// 递归安排下一次决策
			twe.scheduleSingleAIDecision(monsterID)
		})

		if err != nil {
			log.Printf("Failed to schedule AI decision for monster %d: %v", monsterID, err)
			continue
		}

		log.Printf("Scheduled AI decision for monster %d: %s, interval: %v",
			monsterID, taskID, interval)
	}
}

// 单个怪物AI决策 (用于递归调用)
func (twe *TimingWheelExample) scheduleSingleAIDecision(monsterID int) {
	interval := time.Duration(3+rand.Intn(5)) * time.Second

	twe.timingWheel.AddTimer(interval, func() {
		fmt.Printf("[%s] 怪物 %d 再次进行AI决策\n",
			time.Now().Format("15:04:05"), monsterID)
		// 继续递归...
	})
}

// 分钟级任务示例 - 建筑建造
func (twe *TimingWheelExample) scheduleBuildingConstruction() {
	buildings := []struct {
		name     string
		duration time.Duration
	}{
		{"小屋", 2 * time.Minute},
		{"兵营", 5 * time.Minute},
		{"城堡", 15 * time.Minute},
	}

	for _, building := range buildings {
		taskID, err := twe.timingWheel.AddTimer(building.duration, func() {
			fmt.Printf("[%s] 建筑 '%s' 建造完成!\n",
				time.Now().Format("15:04:05"), building.name)
		})

		if err != nil {
			log.Printf("Failed to schedule building construction for %s: %v", building.name, err)
			continue
		}

		log.Printf("Scheduled building construction: %s (%s), duration: %v",
			building.name, taskID, building.duration)
	}
}

// 小时级任务示例 - 玩家状态恢复
func (twe *TimingWheelExample) schedulePlayerRecovery() {
	recoveries := []struct {
		playerID string
		duration time.Duration
	}{
		{"player_001", 1 * time.Hour},
		{"player_002", 2 * time.Hour},
		{"player_003", 6 * time.Hour},
	}

	for _, recovery := range recoveries {
		taskID, err := twe.timingWheel.AddTimer(recovery.duration, func() {
			fmt.Printf("[%s] 玩家 %s 状态完全恢复!\n",
				time.Now().Format("15:04:05"), recovery.playerID)
		})

		if err != nil {
			log.Printf("Failed to schedule player recovery for %s: %v", recovery.playerID, err)
			continue
		}

		log.Printf("Scheduled player recovery: %s (%s), duration: %v",
			recovery.playerID, taskID, recovery.duration)
	}
}

// 天级任务示例 - 日常重置
func (twe *TimingWheelExample) scheduleDailyReset() {
	// 明天0点重置
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	resetTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
		0, 0, 0, 0, tomorrow.Location())
	duration := resetTime.Sub(now)

	taskID, err := twe.timingWheel.AddTimer(duration, func() {
		fmt.Printf("[%s] 每日任务重置!\n",
			time.Now().Format("15:04:05"))

		// 递归安排明天的重置
		twe.scheduleDailyReset()
	})

	if err != nil {
		log.Printf("Failed to schedule daily reset: %v", err)
		return
	}

	log.Printf("Scheduled daily reset: %s, duration: %v", taskID, duration)
}

// 周级任务示例 - 服务器维护
func (twe *TimingWheelExample) scheduleWeeklyMaintenance() {
	// 本周日维护
	now := time.Now()
	daysUntilSunday := (7 - int(now.Weekday())) % 7
	if daysUntilSunday == 0 {
		daysUntilSunday = 7 // 如果今天是周日，下周日
	}

	maintenanceTime := now.AddDate(0, 0, daysUntilSunday)
	maintenanceTime = time.Date(maintenanceTime.Year(), maintenanceTime.Month(),
		maintenanceTime.Day(), 2, 0, 0, 0, maintenanceTime.Location()) // 周日凌晨2点
	duration := maintenanceTime.Sub(now)

	taskID, err := twe.timingWheel.AddTimer(duration, func() {
		fmt.Printf("[%s] 每周服务器维护开始!\n",
			time.Now().Format("15:04:05"))

		// 递归安排下周维护
		twe.scheduleWeeklyMaintenance()
	})

	if err != nil {
		log.Printf("Failed to schedule weekly maintenance: %v", err)
		return
	}

	log.Printf("Scheduled weekly maintenance: %s, duration: %v", taskID, duration)
}

// GetStats 获取统计信息
func (twe *TimingWheelExample) GetStats() map[string]interface{} {
	return twe.timingWheel.GetStats()
}

// Monitor 监控时间轮状态
func (twe *TimingWheelExample) Monitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := twe.GetStats()
		fmt.Printf("\n=== Timing Wheel Stats ===\n")
		fmt.Printf("Running: %v\n", stats["running"])
		fmt.Printf("Total Tasks: %v\n", stats["total_tasks"])
		fmt.Printf("Uptime: %v\n", stats["uptime"])

		for level := 0; level < 6; level++ {
			levelNames := []string{"week", "day", "hour", "minute", "second", "millisecond"}
			if level < len(levelNames) {
				fmt.Printf("%s: %v tasks, %v ticks\n",
					levelNames[level],
					stats[levelNames[level]+"_tasks"],
					stats[levelNames[level]+"_tick_count"])
			}
		}
		fmt.Printf("========================\n")
	}
}
