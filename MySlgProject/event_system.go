package main

import (
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/eventstream"
)

// GameEvent 游戏事件接口
type GameEvent interface {
	GetEventType() string
	GetTimestamp() time.Time
	GetSourceID() string
}

// BaseEvent 基础事件实现
type BaseEvent struct {
	EventType  string
	Timestamp  time.Time
	SourceID   string
	EventData  interface{}
}

// GetEventType 获取事件类型
func (e *BaseEvent) GetEventType() string {
	return e.EventType
}

// GetTimestamp 获取时间戳
func (e *BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetSourceID 获取来源ID
func (e *BaseEvent) GetSourceID() string {
	return e.SourceID
}

// EventSystem 事件系统
type EventSystem struct {
	eventStream    *eventstream.EventStream
	subscriptions  map[string][]*EventSubscription
	mutex          sync.RWMutex
	eventBuffer    []GameEvent
	bufferSize     int
	bufferMutex    sync.RWMutex
	system         *actor.ActorSystem
}

// EventSubscription 事件订阅
type EventSubscription struct {
	ID       string
	Callback EventCallback
	Filter   EventFilter
	Priority int
	Active   bool
}

// EventCallback 事件回调
type EventCallback func(event GameEvent)

// EventFilter 事件过滤器
type EventFilter func(event GameEvent) bool

// NewEventSystem 创建事件系统
func NewEventSystem(system *actor.ActorSystem, bufferSize int) *EventSystem {
	return &EventSystem{
		eventStream:   system.EventStream,
		subscriptions: make(map[string][]*EventSubscription),
		eventBuffer:   make([]GameEvent, 0, bufferSize),
		bufferSize:    bufferSize,
		system:        system,
	}
}

// Publish 发布事件
func (es *EventSystem) Publish(event GameEvent) {
	// 添加到事件流
	es.eventStream.Publish(event)

	// 添加到缓冲区用于历史查询
	es.bufferMutex.Lock()
	es.eventBuffer = append(es.eventBuffer, event)
	if len(es.eventBuffer) > es.bufferSize {
		// 移除最旧的事件
		es.eventBuffer = es.eventBuffer[1:]
	}
	es.bufferMutex.Unlock()

	// 通知订阅者
	es.notifySubscribers(event)
}

// Subscribe 订阅事件
func (es *EventSystem) Subscribe(eventType string, subscription *EventSubscription) string {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	subscription.ID = generateEventSubscriptionID()
	subscription.Active = true

	if _, exists := es.subscriptions[eventType]; !exists {
		es.subscriptions[eventType] = make([]*EventSubscription, 0)
	}

	es.subscriptions[eventType] = append(es.subscriptions[eventType], subscription)

	return subscription.ID
}

// Unsubscribe 取消订阅
func (es *EventSystem) Unsubscribe(eventType, subscriptionID string) bool {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	if subscribers, exists := es.subscriptions[eventType]; exists {
		for i, sub := range subscribers {
			if sub.ID == subscriptionID {
				// 标记为非活跃
				sub.Active = false
				// 从切片中移除
				es.subscriptions[eventType] = append(
					es.subscriptions[eventType][:i],
					es.subscriptions[eventType][i+1:]...,
				)
				return true
			}
		}
	}

	return false
}

// notifySubscribers 通知订阅者
func (es *EventSystem) notifySubscribers(event GameEvent) {
	es.mutex.RLock()
	subscribers := es.subscriptions[event.GetEventType()]
	es.mutex.RUnlock()

	if len(subscribers) == 0 {
		return
	}

	// 按优先级排序
	sortedSubscribers := make([]*EventSubscription, len(subscribers))
	copy(sortedSubscribers, subscribers)

	// 简单冒泡排序：优先级高的先处理
	for i := 0; i < len(sortedSubscribers)-1; i++ {
		for j := i + 1; j < len(sortedSubscribers); j++ {
			if sortedSubscribers[i].Priority < sortedSubscribers[j].Priority {
				sortedSubscribers[i], sortedSubscribers[j] = sortedSubscribers[j], sortedSubscribers[i]
			}
		}
	}

	// 异步通知订阅者
	for _, subscriber := range sortedSubscribers {
		if subscriber.Active && es.matchesFilter(event, subscriber.Filter) {
			go subscriber.Callback(event)
		}
	}
}

// matchesFilter 检查是否匹配过滤器
func (es *EventSystem) matchesFilter(event GameEvent, filter EventFilter) bool {
	if filter == nil {
		return true
	}
	return filter(event)
}

// GetRecentEvents 获取最近事件
func (es *EventSystem) GetRecentEvents(eventType string, limit int) []GameEvent {
	es.bufferMutex.RLock()
	defer es.bufferMutex.RUnlock()

	result := make([]GameEvent, 0)
	count := 0

	// 从缓冲区末尾向前查找（最新的）
	for i := len(es.eventBuffer) - 1; i >= 0 && count < limit; i-- {
		event := es.eventBuffer[i]
		if event.GetEventType() == eventType {
			result = append(result, event)
			count++
		}
	}

	// 反转结果，使其按时间正序排列
	for i := 0; i < len(result)/2; i++ {
		j := len(result) - 1 - i
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// GetEventStats 获取事件统计
func (es *EventSystem) GetEventStats() map[string]interface{} {
	es.mutex.RLock()
	defer es.mutex.RUnlock()

	es.bufferMutex.RLock()
	defer es.bufferMutex.RUnlock()

	stats := make(map[string]interface{})

	// 订阅统计
	totalSubscriptions := 0
	subscriptionBreakdown := make(map[string]int)

	for eventType, subscribers := range es.subscriptions {
		activeCount := 0
		for _, sub := range subscribers {
			if sub.Active {
				activeCount++
			}
		}
		subscriptionBreakdown[eventType] = activeCount
		totalSubscriptions += activeCount
	}

	stats["total_subscriptions"] = totalSubscriptions
	stats["subscription_breakdown"] = subscriptionBreakdown

	// 缓冲区统计
	eventTypeCount := make(map[string]int)
	for _, event := range es.eventBuffer {
		eventTypeCount[event.GetEventType()]++
	}

	stats["buffer_size"] = len(es.eventBuffer)
	stats["max_buffer_size"] = es.bufferSize
	stats["event_type_breakdown"] = eventTypeCount

	return stats
}

// Cleanup 清理无效订阅
func (es *EventSystem) Cleanup() {
	es.mutex.Lock()
	defer es.mutex.Unlock()

	for eventType, subscribers := range es.subscriptions {
		activeSubscribers := make([]*EventSubscription, 0)

		for _, sub := range subscribers {
			if sub.Active {
				activeSubscribers = append(activeSubscribers, sub)
			}
		}

		if len(activeSubscribers) > 0 {
			es.subscriptions[eventType] = activeSubscribers
		} else {
			delete(es.subscriptions, eventType)
		}
	}
}

// CreateEventFilter 创建事件过滤器
func CreateEventFilter(sourceID string, timeRange *TimeRange, customFilter func(GameEvent) bool) EventFilter {
	return func(event GameEvent) bool {
		// 来源ID过滤
		if sourceID != "" && event.GetSourceID() != sourceID {
			return false
		}

		// 时间范围过滤
		if timeRange != nil {
			if event.GetTimestamp().Before(timeRange.Start) ||
			   event.GetTimestamp().After(timeRange.End) {
				return false
			}
		}

		// 自定义过滤
		if customFilter != nil && !customFilter(event) {
			return false
		}

		return true
	}
}

// TimeRange 时间范围
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// CreateTimeRangeFilter 创建时间范围过滤器
func CreateTimeRangeFilter(start, end time.Time) EventFilter {
	return CreateEventFilter("", &TimeRange{Start: start, End: end}, nil)
}

// CreateSourceFilter 创建来源过滤器
func CreateSourceFilter(sourceID string) EventFilter {
	return CreateEventFilter(sourceID, nil, nil)
}

// 具体事件类型定义

// PlayerJoinEvent 玩家加入事件
type PlayerJoinEvent struct {
	BaseEvent
	PlayerID   string
	PlayerName string
	Position   Point
}

// PlayerLeaveEvent 玩家离开事件
type PlayerLeaveEvent struct {
	BaseEvent
	PlayerID string
	Reason   string
}

// EntityMoveEvent 实体移动事件
type EntityMoveEvent struct {
	BaseEvent
	EntityID    string
	FromPos     Point
	ToPos       Point
	EntityType  EntityType
}

// CombatEvent 战斗事件
type CombatEvent struct {
	BaseEvent
	AttackerID   string
	DefenderID   string
	Damage       int
	CombatResult string
	Position     Point
}

// ResourceChangeEvent 资源变更事件
type ResourceChangeEvent struct {
	BaseEvent
	PlayerID    string
	ResourceType ResourceType
	OldAmount   int
	NewAmount   int
	ChangeReason string
}

// BuildingEvent 建筑事件
type BuildingEvent struct {
	BaseEvent
	BuildingID   string
	BuildingType BuildingType
	Action       string // "built", "upgraded", "destroyed"
	PlayerID     string
	Position     Point
}

// GameWorldEvent 游戏世界事件
type GameWorldEvent struct {
	BaseEvent
	EventType string
	EventData interface{}
	Position  Point
}

// 工具函数

// generateEventSubscriptionID 生成订阅ID
func generateEventSubscriptionID() string {
	// 简化实现，实际应该使用更复杂的ID生成算法
	return "sub_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString 生成随机字符串（简化实现）
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

// EventBuilder 事件构建器
type EventBuilder struct {
	event GameEvent
}

// NewEventBuilder 创建事件构建器
func NewEventBuilder(eventType string, sourceID string) *EventBuilder {
	return &EventBuilder{
		event: &BaseEvent{
			EventType: eventType,
			Timestamp: time.Now(),
			SourceID:  sourceID,
		},
	}
}

// WithData 添加数据
func (eb *EventBuilder) WithData(data interface{}) *EventBuilder {
	if baseEvent, ok := eb.event.(*BaseEvent); ok {
		baseEvent.EventData = data
	}
	return eb
}

// WithTimestamp 设置时间戳
func (eb *EventBuilder) WithTimestamp(timestamp time.Time) *EventBuilder {
	if baseEvent, ok := eb.event.(*BaseEvent); ok {
		baseEvent.Timestamp = timestamp
	}
	return eb
}

// Build 构建事件
func (eb *EventBuilder) Build() GameEvent {
	return eb.event
}

// 便捷的构建函数

// NewPlayerJoinEvent 创建玩家加入事件
func NewPlayerJoinEvent(playerID, playerName string, position Point) *PlayerJoinEvent {
	return &PlayerJoinEvent{
		BaseEvent: BaseEvent{
			EventType: "player_join",
			Timestamp: time.Now(),
			SourceID:  playerID,
		},
		PlayerID:   playerID,
		PlayerName: playerName,
		Position:   position,
	}
}

// NewEntityMoveEvent 创建实体移动事件
func NewEntityMoveEvent(entityID string, fromPos, toPos Point, entityType EntityType) *EntityMoveEvent {
	return &EntityMoveEvent{
		BaseEvent: BaseEvent{
			EventType: "entity_move",
			Timestamp: time.Now(),
			SourceID:  entityID,
		},
		EntityID:   entityID,
		FromPos:    fromPos,
		ToPos:      toPos,
		EntityType: entityType,
	}
}

// NewCombatEvent 创建战斗事件
func NewCombatEvent(attackerID, defenderID string, damage int, result string, position Point) *CombatEvent {
	return &CombatEvent{
		BaseEvent: BaseEvent{
			EventType: "combat",
			Timestamp: time.Now(),
			SourceID:  attackerID,
		},
		AttackerID:   attackerID,
		DefenderID:   defenderID,
		Damage:       damage,
		CombatResult: result,
		Position:     position,
	}
}
