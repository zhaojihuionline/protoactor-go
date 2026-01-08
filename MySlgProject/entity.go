package main

import (
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// EntityManager 实体管理器
type EntityManager struct {
	entities   map[string]Entity
	spatialIndex *SpatialIndex
	eventSystem *EventSystem
	mutex       sync.RWMutex
	system      *actor.ActorSystem
}

// NewEntityManager 创建实体管理器
func NewEntityManager(spatialIndex *SpatialIndex, eventSystem *EventSystem, system *actor.ActorSystem) *EntityManager {
	return &EntityManager{
		entities:     make(map[string]Entity),
		spatialIndex: spatialIndex,
		eventSystem: eventSystem,
		system:       system,
	}
}

// AddEntity 添加实体
func (em *EntityManager) AddEntity(entity Entity) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	entityID := entity.GetID()
	if _, exists := em.entities[entityID]; exists {
		return ErrEntityExists
	}

	em.entities[entityID] = entity

	// 添加到空间索引
	em.spatialIndex.InsertEntity(entity)

	// 发布实体创建事件
	em.eventSystem.Publish(NewEntitySpawnEvent(entityID, entity.GetPosition(), entity.GetType()))

	return nil
}

// RemoveEntity 移除实体
func (em *EntityManager) RemoveEntity(entityID string) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	entity, exists := em.entities[entityID]
	if !exists {
		return ErrEntityNotFound
	}

	delete(em.entities, entityID)

	// 从空间索引移除
	em.spatialIndex.RemoveEntity(entityID)

	// 发布实体销毁事件
	em.eventSystem.Publish(NewEntityDespawnEvent(entityID, entity.GetPosition(), entity.GetType()))

	return nil
}

// GetEntity 获取实体
func (em *EntityManager) GetEntity(entityID string) (Entity, bool) {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	entity, exists := em.entities[entityID]
	return entity, exists
}

// UpdateEntityPosition 更新实体位置
func (em *EntityManager) UpdateEntityPosition(entityID string, newPos Point) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	entity, exists := em.entities[entityID]
	if !exists {
		return ErrEntityNotFound
	}

	oldPos := entity.GetPosition()
	entity.SetPosition(newPos)

	// 更新空间索引
	em.spatialIndex.UpdateEntityPosition(entityID, newPos)

	// 发布移动事件
	em.eventSystem.Publish(NewEntityMoveEvent(entityID, oldPos, newPos, entity.GetType()))

	return nil
}

// GetEntitiesInRange 获取范围内的实体
func (em *EntityManager) GetEntitiesInRange(center Point, radius int) []Entity {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	// 使用空间索引进行查询
	return em.spatialIndex.QueryEntitiesInRange(center, radius)
}

// GetEntitiesByType 获取指定类型的实体
func (em *EntityManager) GetEntitiesByType(entityType EntityType) []Entity {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	result := make([]Entity, 0)
	for _, entity := range em.entities {
		if entity.GetType() == entityType {
			result = append(result, entity)
		}
	}

	return result
}

// GetAllEntities 获取所有实体
func (em *EntityManager) GetAllEntities() []Entity {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	result := make([]Entity, 0, len(em.entities))
	for _, entity := range em.entities {
		result = append(result, entity)
	}

	return result
}

// GetEntityCount 获取实体数量
func (em *EntityManager) GetEntityCount() int {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	return len(em.entities)
}

// GetEntityStats 获取实体统计
func (em *EntityManager) GetEntityStats() map[string]interface{} {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	stats := make(map[string]interface{})
	typeCount := make(map[EntityType]int)

	totalEntities := 0
	for _, entity := range em.entities {
		typeCount[entity.GetType()]++
		totalEntities++
	}

	stats["total_entities"] = totalEntities
	stats["type_breakdown"] = typeCount

	return stats
}

// EntityActor 实体Actor包装器
type EntityActor struct {
	entity Entity
	manager *EntityManager
}

// NewEntityActor 创建实体Actor
func NewEntityActor(entity Entity, manager *EntityManager) actor.Producer {
	return func() actor.Actor {
		return &EntityActor{
			entity: entity,
			manager: manager,
		}
	}
}

// Receive 处理消息
func (ea *EntityActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *MoveCommand:
		ea.handleMove(ctx, msg)

	case *AttackCommand:
		ea.handleAttack(ctx, msg)

	case *InteractCommand:
		ea.handleInteract(ctx, msg)

	case *StatusUpdate:
		ea.handleStatusUpdate(ctx, msg)

	case *DamageTaken:
		ea.handleDamage(ctx, msg)

	case *HealReceived:
		ea.handleHeal(ctx, msg)
	}
}

// handleMove 处理移动命令
func (ea *EntityActor) handleMove(ctx actor.Context, cmd *MoveCommand) {
	// 检查移动是否有效
	if !ea.isValidMove(cmd.TargetPos) {
		ctx.Respond(&MoveResult{Success: false, Reason: "invalid_move"})
		return
	}

	// 执行移动
	oldPos := ea.entity.GetPosition()
	ea.manager.UpdateEntityPosition(ea.entity.GetID(), cmd.TargetPos)

	// 响应结果
	ctx.Respond(&MoveResult{
		Success:  true,
		OldPos:   oldPos,
		NewPos:   cmd.TargetPos,
		EntityID: ea.entity.GetID(),
	})
}

// handleAttack 处理攻击命令
func (ea *EntityActor) handleAttack(ctx actor.Context, cmd *AttackCommand) {
	// 查找目标实体
	target, exists := ea.manager.GetEntity(cmd.TargetID)
	if !exists {
		ctx.Respond(&AttackResult{Success: false, Reason: "target_not_found"})
		return
	}

	// 计算伤害
	damage := ea.calculateDamage(target)

	// 应用伤害
	targetPID := ea.getEntityPID(cmd.TargetID)
	if targetPID != nil {
		ea.system.Root.Send(targetPID, &DamageTaken{
			AttackerID: ea.entity.GetID(),
			Damage:     damage,
			DamageType: cmd.DamageType,
		})
	}

	// 发布战斗事件
	ea.manager.eventSystem.Publish(NewCombatEvent(
		ea.entity.GetID(),
		cmd.TargetID,
		damage,
		"hit",
		target.GetPosition(),
	))

	ctx.Respond(&AttackResult{
		Success:    true,
		Damage:     damage,
		TargetID:   cmd.TargetID,
		AttackerID: ea.entity.GetID(),
	})
}

// handleInteract 处理交互命令
func (ea *EntityActor) handleInteract(ctx actor.Context, cmd *InteractCommand) {
	// 处理与建筑、资源的交互
	switch cmd.InteractionType {
	case "harvest":
		ea.handleHarvest(ctx, cmd)
	case "build":
		ea.handleBuild(ctx, cmd)
	case "trade":
		ea.handleTrade(ctx, cmd)
	default:
		ctx.Respond(&InteractResult{Success: false, Reason: "unknown_interaction"})
	}
}

// handleStatusUpdate 处理状态更新
func (ea *EntityActor) handleStatusUpdate(ctx actor.Context, msg *StatusUpdate) {
	// 更新实体状态
	// 这里可以更新实体的属性、状态等
	ctx.Respond(&StatusUpdateAck{EntityID: ea.entity.GetID()})
}

// handleDamage 处理受到伤害
func (ea *EntityActor) handleDamage(ctx actor.Context, msg *DamageTaken) {
	// 计算实际伤害（考虑防御、护盾等）
	actualDamage := ea.calculateActualDamage(msg.Damage, msg.DamageType)

	// 应用伤害效果
	// 这里可以更新实体的生命值、状态效果等

	// 发布伤害事件
	ea.manager.eventSystem.Publish(NewCombatEvent(
		msg.AttackerID,
		ea.entity.GetID(),
		actualDamage,
		"damage_taken",
		ea.entity.GetPosition(),
	))

	ctx.Respond(&DamageResult{
		EntityID:      ea.entity.GetID(),
		DamageTaken:   actualDamage,
		CurrentHealth: 100, // 这里应该从实体状态获取
	})
}

// handleHeal 处理治疗
func (ea *EntityActor) handleHeal(ctx actor.Context, msg *HealReceived) {
	// 应用治疗效果
	// 这里可以更新实体的生命值

	ctx.Respond(&HealResult{
		EntityID:      ea.entity.GetID(),
		HealAmount:    msg.HealAmount,
		CurrentHealth: 100, // 这里应该从实体状态获取
	})
}

// 辅助方法

func (ea *EntityActor) isValidMove(pos Point) bool {
	// 检查移动目标是否有效
	// 这里可以检查地形、碰撞、边界等
	return pos.X >= 0 && pos.X < 1200 && pos.Y >= 0 && pos.Y < 1200
}

func (ea *EntityActor) calculateDamage(target Entity) int {
	// 基础伤害计算
	baseDamage := 10

	// 这里可以根据攻击者属性、目标防御等计算实际伤害
	return baseDamage
}

func (ea *EntityActor) calculateActualDamage(rawDamage int, damageType string) int {
	// 计算实际伤害
	// 这里可以考虑防御、抵抗等
	return rawDamage
}

func (ea *EntityActor) getEntityPID(entityID string) *actor.PID {
	// 这里需要根据实体ID找到对应的Actor PID
	// 实际实现中可能需要一个实体到PID的映射表
	return nil
}

func (ea *EntityActor) handleHarvest(ctx actor.Context, cmd *InteractCommand) {
	// 处理资源采集
	ctx.Respond(&InteractResult{
		Success: true,
		Type:    "harvest",
		Data:    map[string]interface{}{"resource_type": "wood", "amount": 10},
	})
}

func (ea *EntityActor) handleBuild(ctx actor.Context, cmd *InteractCommand) {
	// 处理建筑建造
	ctx.Respond(&InteractResult{
		Success: true,
		Type:    "build",
		Data:    map[string]interface{}{"building_type": "house"},
	})
}

func (ea *EntityActor) handleTrade(ctx actor.Context, cmd *InteractCommand) {
	// 处理交易
	ctx.Respond(&InteractResult{
		Success: true,
		Type:    "trade",
		Data:    map[string]interface{}{"trade_type": "resource_exchange"},
	})
}

// 命令和结果消息定义

type MoveCommand struct {
	TargetPos Point
}

type MoveResult struct {
	Success  bool
	Reason   string
	OldPos   Point
	NewPos   Point
	EntityID string
}

type AttackCommand struct {
	TargetID   string
	DamageType string
}

type AttackResult struct {
	Success    bool
	Reason     string
	Damage     int
	TargetID   string
	AttackerID string
}

type InteractCommand struct {
	InteractionType string
	TargetID        string
	Data            map[string]interface{}
}

type InteractResult struct {
	Success bool
	Reason  string
	Type    string
	Data    interface{}
}

type StatusUpdate struct {
	StatusType string
	Value      interface{}
}

type StatusUpdateAck struct {
	EntityID string
}

type DamageTaken struct {
	AttackerID string
	Damage     int
	DamageType string
}

type DamageResult struct {
	EntityID      string
	DamageTaken   int
	CurrentHealth int
}

type HealReceived struct {
	HealerID   string
	HealAmount int
	HealType   string
}

type HealResult struct {
	EntityID      string
	HealAmount    int
	CurrentHealth int
}

// 错误定义
var (
	ErrEntityExists   = actor.NewError(1, "entity already exists")
	ErrEntityNotFound = actor.NewError(2, "entity not found")
)

// 事件定义

func NewEntitySpawnEvent(entityID string, position Point, entityType EntityType) *EntitySpawnEvent {
	return &EntitySpawnEvent{
		BaseEvent: BaseEvent{
			EventType: "entity_spawn",
			Timestamp: time.Now(),
			SourceID:  entityID,
		},
		EntityID:   entityID,
		Position:   position,
		EntityType: entityType,
	}
}

type EntitySpawnEvent struct {
	BaseEvent
	EntityID   string
	Position   Point
	EntityType EntityType
}

func NewEntityDespawnEvent(entityID string, position Point, entityType EntityType) *EntityDespawnEvent {
	return &EntityDespawnEvent{
		BaseEvent: BaseEvent{
			EventType: "entity_despawn",
			Timestamp: time.Now(),
			SourceID:  entityID,
		},
		EntityID:   entityID,
		Position:   position,
		EntityType: entityType,
	}
}

type EntityDespawnEvent struct {
	BaseEvent
	EntityID   string
	Position   Point
	EntityType EntityType
}
