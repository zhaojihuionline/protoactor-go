package main

import (
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// AOIManager AOI管理器 - 为每个玩家管理可见区域
type AOIManager struct {
	playerID       string
	currentPos     Point
	viewRange      int
	subscriptions  map[string]*Subscription // partitionID -> 订阅信息
	stateCache     map[string]interface{}   // AOI状态缓存
	eventQueue     chan AOIEvent           // AOI事件队列
	cacheMutex     sync.RWMutex
	subMutex       sync.RWMutex
	config         *Config
	system         *actor.ActorSystem
	spatialIndex   *SpatialIndex
	partitionMgr   *PartitionManager
	lastUpdate     time.Time
}

// Subscription AOI订阅信息
type Subscription struct {
	PartitionID string
	ViewRect    Rectangle
	Priority    int
	LastActive  time.Time
}

// AOIManagerActor AOI管理器Actor
type AOIManagerActor struct {
	manager *AOIManager
}

// NewAOIManager 创建AOI管理器
func NewAOIManager(playerID string, initialPos Point, config *Config, system *actor.ActorSystem,
	spatialIndex *SpatialIndex, partitionMgr *PartitionManager) *AOIManager {

	return &AOIManager{
		playerID:     playerID,
		currentPos:   initialPos,
		viewRange:    config.AOI.ViewRange,
		subscriptions: make(map[string]*Subscription),
		stateCache:    make(map[string]interface{}),
		eventQueue:    make(chan AOIEvent, 1000),
		config:        config,
		system:        system,
		spatialIndex:  spatialIndex,
		partitionMgr:  partitionMgr,
		lastUpdate:    time.Now(),
	}
}

// NewAOIManagerActor 创建AOI管理器Actor
func NewAOIManagerActor(manager *AOIManager) actor.Producer {
	return func() actor.Actor {
		return &AOIManagerActor{manager: manager}
	}
}

// Receive 处理消息
func (a *AOIManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *UpdatePosition:
		a.handleUpdatePosition(ctx, msg)

	case *AOIStateChange:
		a.handleStateChange(ctx, msg)

	case *SubscribePartition:
		a.handleSubscribePartition(ctx, msg)

	case *UnsubscribePartition:
		a.handleUnsubscribePartition(ctx, msg)

	case *RequestAOIState:
		a.handleRequestAOIState(ctx, msg)

	case *RefreshAOI:
		a.handleRefreshAOI(ctx)

	case *CleanupAOI:
		a.handleCleanupAOI(ctx)
	}
}

// UpdatePosition 更新玩家位置
func (aoi *AOIManager) UpdatePosition(newPos Point) {
	oldPartitions := aoi.getCoveredPartitions(aoi.currentPos)
	newPartitions := aoi.getCoveredPartitions(newPos)

	// 计算需要取消和新增的订阅
	toUnsubscribe := aoi.getPartitionsToUnsubscribe(oldPartitions, newPartitions)
	toSubscribe := aoi.getPartitionsToSubscribe(oldPartitions, newPartitions)

	// 执行订阅变更
	for partitionID := range toUnsubscribe {
		aoi.unsubscribePartition(partitionID)
	}

	for partitionID := range toSubscribe {
		aoi.subscribePartition(partitionID)
	}

	aoi.currentPos = newPos
	aoi.lastUpdate = time.Now()

	// 发送位置更新事件
	aoi.sendEvent(AOIEvent{
		EventType: AOIEntityMove,
		EntityID:  aoi.playerID,
		Position:  newPos,
		Timestamp: time.Now(),
	})
}

// getCoveredPartitions 获取位置覆盖的分区
func (aoi *AOIManager) getCoveredPartitions(center Point) map[string]bool {
	return aoi.spatialIndex.GetPartitionsInRange(center, aoi.viewRange)
}

// getPartitionsToUnsubscribe 获取需要取消订阅的分区
func (aoi *AOIManager) getPartitionsToUnsubscribe(oldPartitions, newPartitions map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for partitionID := range oldPartitions {
		if !newPartitions[partitionID] {
			result[partitionID] = true
		}
	}
	return result
}

// getPartitionsToSubscribe 获取需要订阅的分区
func (aoi *AOIManager) getPartitionsToSubscribe(oldPartitions, newPartitions map[string]bool) map[string]bool {
	result := make(map[string]bool)
	for partitionID := range newPartitions {
		if !oldPartitions[partitionID] {
			result[partitionID] = true
		}
	}
	return result
}

// subscribePartition 订阅分区
func (aoi *AOIManager) subscribePartition(partitionID string) {
	aoi.subMutex.Lock()
	defer aoi.subMutex.Unlock()

	if _, exists := aoi.subscriptions[partitionID]; !exists {
		viewRect := Rectangle{
			X:      aoi.currentPos.X - aoi.viewRange,
			Y:      aoi.currentPos.Y - aoi.viewRange,
			Width:  aoi.viewRange * 2,
			Height: aoi.viewRange * 2,
		}

		subscription := &Subscription{
			PartitionID: partitionID,
			ViewRect:    viewRect,
			Priority:    1,
			LastActive:  time.Now(),
		}

		aoi.subscriptions[partitionID] = subscription

		// 通知分区管理器
		if partition, exists := aoi.partitionMgr.GetPartition(partitionID); exists {
			// 这里需要获取AOI管理器的PID，暂时用nil代替
			partition.AddSubscriber(aoi.playerID, nil, viewRect)
		}
	}
}

// unsubscribePartition 取消订阅分区
func (aoi *AOIManager) unsubscribePartition(partitionID string) {
	aoi.subMutex.Lock()
	defer aoi.subMutex.Unlock()

	if subscription, exists := aoi.subscriptions[partitionID]; exists {
		delete(aoi.subscriptions, partitionID)

		// 通知分区管理器
		if partition, exists := aoi.partitionMgr.GetPartition(partitionID); exists {
			partition.RemoveSubscriber(aoi.playerID)
		}

		// 清理相关缓存
		aoi.cleanupPartitionCache(partitionID)
	}
}

// GetAOIState 获取AOI状态
func (aoi *AOIManager) GetAOIState() *AOIState {
	aoi.cacheMutex.RLock()
	defer aoi.cacheMutex.RUnlock()

	aoi.subMutex.RLock()
	defer aoi.subMutex.RUnlock()

	state := &AOIState{
		PlayerID:     aoi.playerID,
		CenterPos:    aoi.currentPos,
		ViewRange:    aoi.viewRange,
		VisibleEntities: make([]Entity, 0),
		VisibleGrids:   make([]*GridState, 0),
		LastUpdate:   aoi.lastUpdate,
	}

	// 收集所有可见实体和格子
	for partitionID := range aoi.subscriptions {
		if partition, exists := aoi.partitionMgr.GetPartition(partitionID); exists {
			entities := partition.GetEntitiesInRect(aoi.subscriptions[partitionID].ViewRect)
			state.VisibleEntities = append(state.VisibleEntities, entities...)

			// 这里可以扩展为收集格子状态
		}
	}

	return state
}

// CacheState 缓存状态
func (aoi *AOIManager) CacheState(key string, state interface{}) {
	aoi.cacheMutex.Lock()
	defer aoi.cacheMutex.Unlock()

	aoi.stateCache[key] = state
}

// GetCachedState 获取缓存状态
func (aoi *AOIManager) GetCachedState(key string) (interface{}, bool) {
	aoi.cacheMutex.RLock()
	defer aoi.cacheMutex.RUnlock()

	state, exists := aoi.stateCache[key]
	return state, exists
}

// CleanupExpiredCache 清理过期缓存
func (aoi *AOIManager) CleanupExpiredCache() {
	aoi.cacheMutex.Lock()
	defer aoi.cacheMutex.Unlock()

	now := time.Now()
	for key, state := range aoi.stateCache {
		// 这里可以根据状态的时间戳判断是否过期
		// 暂时清理超过5分钟未访问的缓存
		if now.Sub(aoi.lastUpdate) > 5*time.Minute {
			delete(aoi.stateCache, key)
		}
	}
}

// sendEvent 发送AOI事件
func (aoi *AOIManager) sendEvent(event AOIEvent) {
	select {
	case aoi.eventQueue <- event:
		// 事件已入队
	default:
		// 队列满，丢弃或异步处理
		go func() {
			select {
			case aoi.eventQueue <- event:
			default:
				// 仍然无法入队，记录警告
			}
		}()
	}
}

// cleanupPartitionCache 清理分区缓存
func (aoi *AOIManager) cleanupPartitionCache(partitionID string) {
	aoi.cacheMutex.Lock()
	defer aoi.cacheMutex.Unlock()

	// 清理以分区ID为前缀的缓存
	prefix := partitionID + "_"
	for key := range aoi.stateCache {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(aoi.stateCache, key)
		}
	}
}

// Handle message handlers

func (a *AOIManagerActor) handleUpdatePosition(ctx actor.Context, msg *UpdatePosition) {
	a.manager.UpdatePosition(msg.NewPosition)

	// 回复确认
	ctx.Respond(&PositionUpdateAck{
		PlayerID:   a.manager.playerID,
		NewPosition: msg.NewPosition,
		Timestamp:  time.Now(),
	})
}

func (a *AOIManagerActor) handleStateChange(ctx actor.Context, msg *AOIStateChange) {
	// 处理状态变更
	cacheKey := msg.PartitionID + "_state_" + msg.Change.GridPos.String()
	a.manager.CacheState(cacheKey, msg.Change)

	// 发送给客户端（这里应该是转发给玩家的连接）
	a.sendToClient(msg)
}

func (a *AOIManagerActor) handleSubscribePartition(ctx actor.Context, msg *SubscribePartition) {
	a.manager.subscribePartition(msg.PartitionID)
	ctx.Respond(&SubscribeAck{PartitionID: msg.PartitionID})
}

func (a *AOIManagerActor) handleUnsubscribePartition(ctx actor.Context, msg *UnsubscribePartition) {
	a.manager.unsubscribePartition(msg.PartitionID)
	ctx.Respond(&UnsubscribeAck{PartitionID: msg.PartitionID})
}

func (a *AOIManagerActor) handleRequestAOIState(ctx actor.Context, msg *RequestAOIState) {
	state := a.manager.GetAOIState()
	ctx.Respond(state)
}

func (a *AOIManagerActor) handleRefreshAOI(ctx actor.Context) {
	// 强制刷新AOI状态
	a.manager.CleanupExpiredCache()

	// 重新评估订阅
	currentPartitions := a.manager.getCoveredPartitions(a.manager.currentPos)
	a.manager.subMutex.RLock()
	activeSubscriptions := make(map[string]bool)
	for partitionID := range a.manager.subscriptions {
		activeSubscriptions[partitionID] = true
	}
	a.manager.subMutex.RUnlock()

	// 取消无效订阅
	for partitionID := range activeSubscriptions {
		if !currentPartitions[partitionID] {
			a.manager.unsubscribePartition(partitionID)
		}
	}

	// 订阅缺失分区
	for partitionID := range currentPartitions {
		if !activeSubscriptions[partitionID] {
			a.manager.subscribePartition(partitionID)
		}
	}

	ctx.Respond(&RefreshAck{Timestamp: time.Now()})
}

func (a *AOIManagerActor) handleCleanupAOI(ctx actor.Context) {
	a.manager.CleanupExpiredCache()
	ctx.Respond(&CleanupAck{Timestamp: time.Now()})
}

func (a *AOIManagerActor) sendToClient(msg *AOIStateChange) {
	// 这里应该是发送给客户端的逻辑
	// 实际实现中会通过网络连接发送
}

// AOIState AOI状态
type AOIState struct {
	PlayerID       string
	CenterPos      Point
	ViewRange      int
	VisibleEntities []Entity
	VisibleGrids   []*GridState
	LastUpdate     time.Time
}

// Message types for AOI communication
type UpdatePosition struct {
	NewPosition Point
}

type PositionUpdateAck struct {
	PlayerID    string
	NewPosition Point
	Timestamp   time.Time
}

type SubscribePartition struct {
	PartitionID string
}

type SubscribeAck struct {
	PartitionID string
}

type UnsubscribePartition struct {
	PartitionID string
}

type UnsubscribeAck struct {
	PartitionID string
}

type RequestAOIState struct{}

type RefreshAOI struct{}

type RefreshAck struct {
	Timestamp time.Time
}

type CleanupAOI struct{}

type CleanupAck struct {
	Timestamp time.Time
}
