package core

import (
	"sync"

	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

// EntityManager 实体管理器，负责管理游戏中的所有实体
type EntityManager struct {
	entities map[bmap.EntityID]*bmap.Entity
	mu       sync.RWMutex
}

// NewEntityManager 创建新的实体管理器
func NewEntityManager() *EntityManager {
	return &EntityManager{
		entities: make(map[bmap.EntityID]*bmap.Entity),
	}
}

// AddEntity 添加实体到管理器
func (em *EntityManager) AddEntity(entity *bmap.Entity) error {
	if entity == nil {
		return nil // 忽略空实体
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	em.entities[bmap.EntityID(entity.ID)] = entity
	return nil
}

// RemoveEntity 从管理器中移除实体
func (em *EntityManager) RemoveEntity(entityID bmap.EntityID) {
	em.mu.Lock()
	defer em.mu.Unlock()

	delete(em.entities, entityID)
}

// GetEntity 根据ID获取实体
func (em *EntityManager) GetEntity(entityID bmap.EntityID) (*bmap.Entity, bool) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	entity, exists := em.entities[entityID]
	return entity, exists
}

// UpdateEntity 更新实体信息
func (em *EntityManager) UpdateEntity(entity *bmap.Entity) error {
	if entity == nil {
		return nil
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	em.entities[bmap.EntityID(entity.ID)] = entity
	return nil
}

// GetEntitiesInArea 获取指定矩形区域内的所有实体
func (em *EntityManager) GetEntitiesInArea(minX, minY, maxX, maxY int32) []*bmap.Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var entities []*bmap.Entity
	for _, entity := range em.entities {
		if entity.Position != nil {
			if entity.Position.X >= minX && entity.Position.X < maxX &&
				entity.Position.Y >= minY && entity.Position.Y < maxY {
				entities = append(entities, entity)
			}
		}
	}
	return entities
}

// GetEntitiesByType 根据实体类型获取所有实体
func (em *EntityManager) GetEntitiesByType(entityType bmap.EntityType) []*bmap.Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var entities []*bmap.Entity
	for _, entity := range em.entities {
		if entity.Type == entityType {
			entities = append(entities, entity)
		}
	}
	return entities
}

// GetAllEntities 获取所有实体
func (em *EntityManager) GetAllEntities() []*bmap.Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()

	entities := make([]*bmap.Entity, 0, len(em.entities))
	for _, entity := range em.entities {
		entities = append(entities, entity)
	}
	return entities
}

// Count 返回实体总数
func (em *EntityManager) Count() int {
	em.mu.RLock()
	defer em.mu.RUnlock()

	return len(em.entities)
}

// Clear 清空所有实体
func (em *EntityManager) Clear() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.entities = make(map[bmap.EntityID]*bmap.Entity)
}

// HasEntity 检查实体是否存在
func (em *EntityManager) HasEntity(entityID bmap.EntityID) bool {
	em.mu.RLock()
	defer em.mu.RUnlock()

	_, exists := em.entities[entityID]
	return exists
}

// GetEntitiesByPosition 获取指定位置的所有实体
func (em *EntityManager) GetEntitiesByPosition(x, y int32) []*bmap.Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var entities []*bmap.Entity
	for _, entity := range em.entities {
		if entity.Position != nil && entity.Position.X == x && entity.Position.Y == y {
			entities = append(entities, entity)
		}
	}
	return entities
}

// UpdateEntityPosition 更新实体位置
func (em *EntityManager) UpdateEntityPosition(entityID bmap.EntityID, x, y int32) bool {
	em.mu.Lock()
	defer em.mu.Unlock()

	if entity, exists := em.entities[entityID]; exists {
		if entity.Position == nil {
			entity.Position = &bmap.Position{}
		}
		entity.Position.X = x
		entity.Position.Y = y
		entity.Version++ // 位置变更时版本递增
		return true
	}
	return false
}

// GetEntitiesWithVersionGreaterThan 获取版本号大于指定值的实体
func (em *EntityManager) GetEntitiesWithVersionGreaterThan(minVersion int64) []*bmap.Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var entities []*bmap.Entity
	for _, entity := range em.entities {
		if entity.Version > minVersion {
			entities = append(entities, entity)
		}
	}
	return entities
}

// GetEntityVersion 获取实体版本号
func (em *EntityManager) GetEntityVersion(entityID bmap.EntityID) (int64, bool) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if entity, exists := em.entities[entityID]; exists {
		return entity.Version, true
	}
	return 0, false
}

// UpdateEntityVersion 更新实体版本号
func (em *EntityManager) UpdateEntityVersion(entityID bmap.EntityID) bool {
	em.mu.Lock()
	defer em.mu.Unlock()

	if entity, exists := em.entities[entityID]; exists {
		entity.Version++
		return true
	}
	return false
}
