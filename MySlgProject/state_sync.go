package main

import (
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
)

// StateSyncManager 状态同步管理器
type StateSyncManager struct {
	partitionMgr   *PartitionManager
	eventStream    *eventstream.EventStream
	subscribers    map[string][]*SyncSubscriber // 分区ID -> 订阅者列表
	changeQueue    chan *StateChange
	batchSize      int
	batchTimeout   time.Duration
	mutex          sync.RWMutex
	system         *actor.ActorSystem
}

// SyncSubscriber 同步订阅者
type SyncSubscriber struct {
	ID           string
	Callback     StateChangeCallback
	LastVersion  int64
	Filter       StateFilter
	Priority     int
}

// StateChangeCallback 状态变更回调
type StateChangeCallback func(changes []*StateChange)

// StateFilter 状态过滤器
type StateFilter func(change *StateChange) bool

// SyncMode 同步模式
type SyncMode int

const (
	SyncModeRealtime SyncMode = iota // 实时同步
	SyncModeBatched                  // 批量同步
	SyncModePeriodic                 // 定期同步
)

// NewStateSyncManager 创建状态同步管理器
func NewStateSyncManager(partitionMgr *PartitionManager, system *actor.ActorSystem,
	batchSize int, batchTimeout time.Duration) *StateSyncManager {

	sm := &StateSyncManager{
		partitionMgr: partitionMgr,
		eventStream:  system.EventStream,
		subscribers:  make(map[string][]*SyncSubscriber),
		changeQueue:  make(chan *StateChange, 10000),
		batchSize:    batchSize,
		batchTimeout: batchTimeout,
		system:       system,
	}

	// 订阅分区状态变更事件
	sm.subscribeToPartitionEvents()

	// 启动批量处理
	go sm.batchProcessor()

	return sm
}

// Subscribe 订阅状态变更
func (sm *StateSyncManager) Subscribe(partitionID string, subscriber *SyncSubscriber) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if _, exists := sm.subscribers[partitionID]; !exists {
		sm.subscribers[partitionID] = make([]*SyncSubscriber, 0)
	}

	sm.subscribers[partitionID] = append(sm.subscribers[partitionID], subscriber)
}

// Unsubscribe 取消订阅
func (sm *StateSyncManager) Unsubscribe(partitionID, subscriberID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if subscribers, exists := sm.subscribers[partitionID]; exists {
		for i, sub := range subscribers {
			if sub.ID == subscriberID {
				// 移除订阅者
				sm.subscribers[partitionID] = append(
					sm.subscribers[partitionID][:i],
					sm.subscribers[partitionID][i+1:]...,
				)
				break
			}
		}
	}
}

// PublishChange 发布状态变更
func (sm *StateSyncManager) PublishChange(change *StateChange) {
	// 更新版本向量
	sm.mutex.Lock()
	sm.versionVector.PartitionVersions[change.GridPos.String()] = change.Version
	sm.versionVector.Timestamp = time.Now()
	sm.mutex.Unlock()

	// 发送到变更队列
	select {
	case sm.changeQueue <- change:
		// 成功入队
	default:
		// 队列满，异步处理
		go func() {
			select {
			case sm.changeQueue <- change:
			default:
				// 仍然无法入队，记录警告
			}
		}()
	}
}

// GetStateSnapshot 获取状态快照
func (sm *StateSyncManager) GetStateSnapshot(partitionID string, sinceVersion int64) []*StateChange {
	partition := sm.partitionMgr.GetPartition(partitionID)
	if partition == nil {
		return nil
	}

	changes := make([]*StateChange, 0)

	// 获取分区内所有状态的变更
	for pos, state := range partition.GridStates {
		if state.Version > sinceVersion {
			change := &StateChange{
				GridPos:    pos,
				ChangeType: ChangeTerrain, // 简化处理
				NewState:   state,
				Timestamp:  state.LastUpdate,
				Version:    state.Version,
			}
			changes = append(changes, change)
		}
	}

	return changes
}

// SyncToVersion 同步到指定版本
func (sm *StateSyncManager) SyncToVersion(partitionID string, targetVersion int64, subscriberID string) {
	changes := sm.GetStateSnapshot(partitionID, 0) // 获取所有变更

	// 过滤到目标版本
	filteredChanges := make([]*StateChange, 0)
	for _, change := range changes {
		if change.Version <= targetVersion {
			filteredChanges = append(filteredChanges, change)
		}
	}

	// 发送给订阅者
	if subscribers, exists := sm.subscribers[partitionID]; exists {
		for _, subscriber := range subscribers {
			if subscriber.ID == subscriberID {
				go subscriber.Callback(filteredChanges)
				break
			}
		}
	}
}

// batchProcessor 批量处理器
func (sm *StateSyncManager) batchProcessor() {
	batch := make([]*StateChange, 0, sm.batchSize)
	timer := time.NewTimer(sm.batchTimeout)
	defer timer.Stop()

	for {
		select {
		case change := <-sm.changeQueue:
			batch = append(batch, change)

			if len(batch) >= sm.batchSize {
				sm.processBatch(batch)
				batch = batch[:0]
				timer.Reset(sm.batchTimeout)
			}

		case <-timer.C:
			if len(batch) > 0 {
				sm.processBatch(batch)
				batch = batch[:0]
			}
			timer.Reset(sm.batchTimeout)
		}
	}
}

// processBatch 处理批量变更
func (sm *StateSyncManager) processBatch(changes []*StateChange) {
	// 按分区分组
	partitionChanges := make(map[string][]*StateChange)

	for _, change := range changes {
		partitionID := sm.getPartitionIDForChange(change)
		if _, exists := partitionChanges[partitionID]; !exists {
			partitionChanges[partitionID] = make([]*StateChange, 0)
		}
		partitionChanges[partitionID] = append(partitionChanges[partitionID], change)
	}

	// 为每个分区处理变更
	for partitionID, changes := range partitionChanges {
		sm.processPartitionChanges(partitionID, changes)
	}
}

// processPartitionChanges 处理分区变更
func (sm *StateSyncManager) processPartitionChanges(partitionID string, changes []*StateChange) {
	sm.mutex.RLock()
	subscribers := sm.subscribers[partitionID]
	sm.mutex.RUnlock()

	if len(subscribers) == 0 {
		return
	}

	// 按优先级排序订阅者
	sortedSubscribers := make([]*SyncSubscriber, len(subscribers))
	copy(sortedSubscribers, subscribers)

	// 简单排序：优先级高的先处理
	for i := 0; i < len(sortedSubscribers)-1; i++ {
		for j := i + 1; j < len(sortedSubscribers); j++ {
			if sortedSubscribers[i].Priority < sortedSubscribers[j].Priority {
				sortedSubscribers[i], sortedSubscribers[j] = sortedSubscribers[j], sortedSubscribers[i]
			}
		}
	}

	// 分发变更给订阅者
	for _, subscriber := range sortedSubscribers {
		filteredChanges := sm.filterChanges(changes, subscriber.Filter)
		if len(filteredChanges) > 0 {
			go subscriber.Callback(filteredChanges)
		}
	}
}

// filterChanges 过滤变更
func (sm *StateSyncManager) filterChanges(changes []*StateChange, filter StateFilter) []*StateChange {
	if filter == nil {
		return changes
	}

	filtered := make([]*StateChange, 0)
	for _, change := range changes {
		if filter(change) {
			filtered = append(filtered, change)
		}
	}

	return filtered
}

// getPartitionIDForChange 获取变更对应的分区ID
func (sm *StateSyncManager) getPartitionIDForChange(change *StateChange) string {
	return sm.partitionMgr.spatialIndex.GetPartitionAt(change.GridPos)
}

// subscribeToPartitionEvents 订阅分区事件
func (sm *StateSyncManager) subscribeToPartitionEvents() {
	// 订阅分区状态变更事件
	sm.eventStream.SubscribeWithPredicate(
		func(event interface{}) {
			if change, ok := event.(*StateChange); ok {
				sm.PublishChange(change)
			}
		},
		func(event interface{}) bool {
			_, ok := event.(*StateChange)
			return ok
		},
	)
}


// GetSyncStats 获取同步统计
func (sm *StateSyncManager) GetSyncStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	totalSubscribers := 0
	subscriberBreakdown := make(map[string]int)

	for partitionID, subscribers := range sm.subscribers {
		count := len(subscribers)
		totalSubscribers += count
		subscriberBreakdown[partitionID] = count
	}

	return map[string]interface{}{
		"total_subscribers":     totalSubscribers,
		"partition_breakdown":   subscriberBreakdown,
		"queue_size":           len(sm.changeQueue),
		"batch_size":           sm.batchSize,
		"batch_timeout":        sm.batchTimeout.String(),
		"version_vector_size":  len(sm.versionVector.PartitionVersions),
		"last_update":          sm.versionVector.Timestamp,
	}
}

// CreateStateFilter 创建状态过滤器
func CreateStateFilter(changeTypes []ChangeType, positionFilter func(Point) bool) StateFilter {
	return func(change *StateChange) bool {
		// 检查变更类型
		typeAllowed := false
		for _, allowedType := range changeTypes {
			if change.ChangeType == allowedType {
				typeAllowed = true
				break
			}
		}
		if !typeAllowed {
			return false
		}

		// 检查位置
		if positionFilter != nil && !positionFilter(change.GridPos) {
			return false
		}

		return true
	}
}

// CreateDistanceFilter 创建距离过滤器
func CreateDistanceFilter(center Point, maxDistance int) func(Point) bool {
	return func(pos Point) bool {
		return center.Distance(pos) <= float64(maxDistance*maxDistance)
	}
}

// CreateRectangleFilter 创建矩形过滤器
func CreateRectangleFilter(rect Rectangle) func(Point) bool {
	return func(pos Point) bool {
		return rect.Contains(pos)
	}
}
