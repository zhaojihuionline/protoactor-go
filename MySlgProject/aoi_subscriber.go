package main

import (
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// AOISubscriber AOI订阅管理器
type AOISubscriber struct {
	playerID       string
	aoiManager     *actor.PID
	subscriptions  map[string]*PartitionSubscription
	eventBuffer    []*AOIEvent
	bufferMutex    sync.RWMutex
	config         *Config
	system         *actor.ActorSystem
	partitionMgr   *PartitionManager
	lastCleanup    time.Time
}

// PartitionSubscription 分区订阅信息
type PartitionSubscription struct {
	PartitionID string
	ViewRect    Rectangle
	Priority    int
	State       SubscriptionState
	LastUpdate  time.Time
	EventCount  int64
}

// SubscriptionState 订阅状态
type SubscriptionState int

const (
	SubStateActive SubscriptionState = iota
	SubStateInactive
	SubStatePending
	SubStateError
)

// NewAOISubscriber 创建AOI订阅器
func NewAOISubscriber(playerID string, aoiManager *actor.PID, config *Config,
	system *actor.ActorSystem, partitionMgr *PartitionManager) *AOISubscriber {

	return &AOISubscriber{
		playerID:      playerID,
		aoiManager:    aoiManager,
		subscriptions: make(map[string]*PartitionSubscription),
		eventBuffer:   make([]*AOIEvent, 0),
		config:        config,
		system:        system,
		partitionMgr:  partitionMgr,
		lastCleanup:   time.Now(),
	}
}

// SubscribePartition 订阅分区
func (sub *AOISubscriber) SubscribePartition(partitionID string, viewRect Rectangle) error {
	sub.bufferMutex.Lock()
	defer sub.bufferMutex.Unlock()

	if _, exists := sub.subscriptions[partitionID]; exists {
		// 已经订阅，更新视图
		sub.subscriptions[partitionID].ViewRect = viewRect
		sub.subscriptions[partitionID].LastUpdate = time.Now()
		return nil
	}

	// 创建新订阅
	subscription := &PartitionSubscription{
		PartitionID: partitionID,
		ViewRect:    viewRect,
		Priority:    1,
		State:       SubStatePending,
		LastUpdate:  time.Now(),
		EventCount:  0,
	}

	sub.subscriptions[partitionID] = subscription

	// 通知分区管理器
	if partition, exists := sub.partitionMgr.GetPartition(partitionID); exists {
		partition.AddSubscriber(sub.playerID, sub.aoiManager, viewRect)
		subscription.State = SubStateActive
	} else {
		subscription.State = SubStateError
		return ErrPartitionNotFound
	}

	return nil
}

// UnsubscribePartition 取消订阅分区
func (sub *AOISubscriber) UnsubscribePartition(partitionID string) error {
	sub.bufferMutex.Lock()
	defer sub.bufferMutex.Unlock()

	if _, exists := sub.subscriptions[partitionID]; exists {
		// 通知分区管理器
		if partition, exists := sub.partitionMgr.GetPartition(partitionID); exists {
			partition.RemoveSubscriber(sub.playerID)
		}

		delete(sub.subscriptions, partitionID)

		// 清理相关事件缓冲
		sub.cleanupPartitionEvents(partitionID)
	}

	return nil
}

// UpdateSubscription 更新订阅视图
func (sub *AOISubscriber) UpdateSubscription(partitionID string, viewRect Rectangle) error {
	sub.bufferMutex.Lock()
	defer sub.bufferMutex.Unlock()

	if _, exists := sub.subscriptions[partitionID]; exists {
		oldRect := subscription.ViewRect
		subscription.ViewRect = viewRect
		subscription.LastUpdate = time.Now()

		// 通知分区管理器更新视图
		if partition, exists := sub.partitionMgr.GetPartition(partitionID); exists {
			partition.UpdateSubscriberView(sub.playerID, viewRect)
		}

		// 如果视图变化较大，可能需要重新同步状态
		if sub.shouldResync(oldRect, viewRect) {
			sub.requestResync(partitionID)
		}
	}

	return nil
}

// HandleAOIEvent 处理AOI事件
func (sub *AOISubscriber) HandleAOIEvent(event *AOIEvent) {
	sub.bufferMutex.Lock()
	defer sub.bufferMutex.Unlock()

	// 添加到事件缓冲
	sub.eventBuffer = append(sub.eventBuffer, event)

	// 更新订阅统计
	if subscription, exists := sub.subscriptions[event.EntityID]; exists {
		subscription.EventCount++
		subscription.LastUpdate = time.Now()
	}

	// 检查是否需要批量处理
	if len(sub.eventBuffer) >= sub.config.AOI.BatchSize {
		sub.processBatchEvents()
	}
}

// GetBufferedEvents 获取缓冲的事件
func (sub *AOISubscriber) GetBufferedEvents() []*AOIEvent {
	sub.bufferMutex.RLock()
	defer sub.bufferMutex.RUnlock()

	events := make([]*AOIEvent, len(sub.eventBuffer))
	copy(events, sub.eventBuffer)
	return events
}

// ClearBuffer 清空事件缓冲
func (sub *AOISubscriber) ClearBuffer() {
	sub.bufferMutex.Lock()
	defer sub.bufferMutex.Unlock()

	sub.eventBuffer = sub.eventBuffer[:0]
}

// GetSubscriptionStats 获取订阅统计信息
func (sub *AOISubscriber) GetSubscriptionStats() map[string]interface{} {
	sub.bufferMutex.RLock()
	defer sub.bufferMutex.RUnlock()

	stats := make(map[string]interface{})
	stats["player_id"] = sub.playerID
	stats["subscription_count"] = len(sub.subscriptions)
	stats["buffered_events"] = len(sub.eventBuffer)
	stats["last_cleanup"] = sub.lastCleanup

	subscriptionStats := make(map[string]interface{})
	for partitionID, subscription := range sub.subscriptions {
		subStats := map[string]interface{}{
			"state":       subscription.State,
			"event_count": subscription.EventCount,
			"last_update": subscription.LastUpdate,
			"view_rect":   subscription.ViewRect,
		}
		subscriptionStats[partitionID] = subStats
	}
	stats["subscriptions"] = subscriptionStats

	return stats
}

// Cleanup 清理过期订阅和缓冲
func (sub *AOISubscriber) Cleanup() {
	sub.bufferMutex.Lock()
	defer sub.bufferMutex.Unlock()

	now := time.Now()

	// 清理过期订阅（超过5分钟未更新）
	for partitionID, subscription := range sub.subscriptions {
		if now.Sub(subscription.LastUpdate) > 5*time.Minute {
			delete(sub.subscriptions, partitionID)
			sub.cleanupPartitionEvents(partitionID)
		}
	}

	// 清理过期事件缓冲（超过1分钟）
	if len(sub.eventBuffer) > 0 {
		validEvents := make([]*AOIEvent, 0)
		for _, event := range sub.eventBuffer {
			if now.Sub(event.Timestamp) <= time.Minute {
				validEvents = append(validEvents, event)
			}
		}
		sub.eventBuffer = validEvents
	}

	sub.lastCleanup = now
}

// processBatchEvents 批量处理事件
func (sub *AOISubscriber) processBatchEvents() {
	if len(sub.eventBuffer) == 0 {
		return
	}

	// 合并相同类型的连续事件
	mergedEvents := sub.mergeEvents(sub.eventBuffer)

	// 发送批量事件到AOI管理器
	batchEvent := &AOIBatchEvent{
		PlayerID: sub.playerID,
		Events:   mergedEvents,
		Timestamp: time.Now(),
	}

	sub.system.Root.Send(sub.aoiManager, batchEvent)

	// 清空缓冲
	sub.eventBuffer = sub.eventBuffer[:0]
}

// mergeEvents 合并事件
func (sub *AOISubscriber) mergeEvents(events []*AOIEvent) []*AOIEvent {
	if len(events) <= 1 {
		return events
	}

	// 简单的合并策略：合并相同实体ID的连续移动事件
	merged := make([]*AOIEvent, 0)
	current := events[0]

	for i := 1; i < len(events); i++ {
		event := events[i]

		// 如果是同一个实体的连续移动事件，可以合并
		if current.EventType == AOIEntityMove && event.EventType == AOIEntityMove &&
		   current.EntityID == event.EntityID {

			// 更新当前位置
			current.Position = event.Position
			current.Timestamp = event.Timestamp
		} else {
			// 无法合并，添加当前事件并开始新的事件
			merged = append(merged, current)
			current = event
		}
	}

	// 添加最后一个事件
	merged = append(merged, current)

	return merged
}

// cleanupPartitionEvents 清理分区相关事件
func (sub *AOISubscriber) cleanupPartitionEvents(partitionID string) {
	validEvents := make([]*AOIEvent, 0)

	for _, event := range sub.eventBuffer {
		// 这里可以根据事件内容判断是否与分区相关
		// 暂时保留所有事件
		validEvents = append(validEvents, event)
	}

	sub.eventBuffer = validEvents
}

// shouldResync 判断是否需要重新同步
func (sub *AOISubscriber) shouldResync(oldRect, newRect Rectangle) bool {
	// 计算视图变化的程度
	oldArea := oldRect.Width * oldRect.Height
	newArea := newRect.Width * newRect.Height

	// 如果新视图超出旧视图太多，需要重新同步
	if newArea > oldArea*2 {
		return true
	}

	// 如果中心点移动距离超过视图范围的1/4
	centerDist := oldRect.X+oldRect.Width/2 - newRect.X+newRect.Width/2
	if centerDist*centerDist > (sub.config.AOI.ViewRange*sub.config.AOI.ViewRange)/16 {
		return true
	}

	return false
}

// requestResync 请求重新同步
func (sub *AOISubscriber) requestResync(partitionID string) {
	resyncMsg := &RequestPartitionResync{
		PlayerID:    sub.playerID,
		PartitionID: partitionID,
	}

	if partition, exists := sub.partitionMgr.GetPartition(partitionID); exists {
		// 发送重新同步请求到分区
		sub.system.Root.Send(partition.ActorPID, resyncMsg)
	}
}

// AOIBatchEvent AOI批量事件
type AOIBatchEvent struct {
	PlayerID  string
	Events    []*AOIEvent
	Timestamp time.Time
}

// RequestPartitionResync 请求分区重新同步
type RequestPartitionResync struct {
	PlayerID    string
	PartitionID string
}

// Errors
var (
	ErrPartitionNotFound = errors.New("partition not found")
)
