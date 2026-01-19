package core

import (
	"sync"

	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

// EntityManager 实体管理器，负责管理游戏中的所有实体（按层级组织）
type EntityManager struct {
	entitiesByLayer map[bmap.LayerNumber]*map[bmap.EntityID]*bmap.Entity
	mu              sync.RWMutex
}

// NewEntityManager 创建新的实体管理器
func NewEntityManager() *EntityManager {
	return &EntityManager{
		entitiesByLayer: make(map[bmap.LayerNumber]*map[bmap.EntityID]*bmap.Entity),
	}
}

// ensureLayerMap 确保指定层级的实体映射存在
func (em *EntityManager) ensureLayerMap(layer bmap.LayerNumber) *map[bmap.EntityID]*bmap.Entity {
	if em.entitiesByLayer[layer] == nil {
		em.entitiesByLayer[layer] = &map[bmap.EntityID]*bmap.Entity{}
	}
	return em.entitiesByLayer[layer]
}

// AddEntity 添加实体到指定层级
func (em *EntityManager) AddEntity(layer bmap.LayerNumber, entity *bmap.Entity) error {
	if entity == nil {
		return nil // 忽略空实体
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	layerMap := em.ensureLayerMap(layer)
	(*layerMap)[bmap.EntityID(entity.ID)] = entity
	return nil
}

// RemoveEntity 从指定层级中移除实体
func (em *EntityManager) RemoveEntity(layer bmap.LayerNumber, entityID bmap.EntityID) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if layerMap, exists := em.entitiesByLayer[layer]; exists {
		delete(*layerMap, entityID)
		// 如果该层级没有实体了，可以选择清理空的映射
		if len(*layerMap) == 0 {
			delete(em.entitiesByLayer, layer)
		}
	}
}

// GetEntity 从指定层级获取实体
func (em *EntityManager) GetEntity(layer bmap.LayerNumber, entityID bmap.EntityID) (*bmap.Entity, bool) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if layerMap, exists := em.entitiesByLayer[layer]; exists {
		entity, found := (*layerMap)[entityID]
		return entity, found
	}
	return nil, false
}

// UpdateEntity 更新指定层级的实体信息
func (em *EntityManager) UpdateEntity(layer bmap.LayerNumber, entity *bmap.Entity) error {
	if entity == nil {
		return nil
	}

	em.mu.Lock()
	defer em.mu.Unlock()

	layerMap := em.ensureLayerMap(layer)
	(*layerMap)[bmap.EntityID(entity.ID)] = entity
	return nil
}

// GetEntitiesInArea 获取指定层级和区域内的所有实体
func (em *EntityManager) GetEntitiesInArea(layer bmap.LayerNumber, minX, minY, maxX, maxY int32) []*bmap.Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var entities []*bmap.Entity
	if layerMap, exists := em.entitiesByLayer[layer]; exists {
		for _, entity := range *layerMap {
			if entity.Position != nil {
				if entity.Position.X >= minX && entity.Position.X < maxX &&
					entity.Position.Y >= minY && entity.Position.Y < maxY {
					entities = append(entities, entity)
				}
			}
		}
	}
	return entities
}

// GetEntitiesByType 获取指定层级和类型的实体
func (em *EntityManager) GetEntitiesByType(layer bmap.LayerNumber, entityType bmap.EntityType) []*bmap.Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var entities []*bmap.Entity
	if layerMap, exists := em.entitiesByLayer[layer]; exists {
		for _, entity := range *layerMap {
			if entity.Type == entityType {
				entities = append(entities, entity)
			}
		}
	}
	return entities
}

// GetAllEntitiesInLayer 获取指定层级的所有实体
func (em *EntityManager) GetAllEntitiesInLayer(layer bmap.LayerNumber) []*bmap.Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if layerMap, exists := em.entitiesByLayer[layer]; exists {
		entities := make([]*bmap.Entity, 0, len(*layerMap))
		for _, entity := range *layerMap {
			entities = append(entities, entity)
		}
		return entities
	}
	return []*bmap.Entity{}
}

// GetAllEntities 获取所有层级的所有实体
func (em *EntityManager) GetAllEntities() []*bmap.Entity {
	em.mu.RLock()
	defer em.mu.RUnlock()

	var allEntities []*bmap.Entity
	for _, layerMap := range em.entitiesByLayer {
		for _, entity := range *layerMap {
			allEntities = append(allEntities, entity)
		}
	}
	return allEntities
}

// Count 返回指定层级的实体总数
func (em *EntityManager) Count(layer bmap.LayerNumber) int {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if layerMap, exists := em.entitiesByLayer[layer]; exists {
		return len(*layerMap)
	}
	return 0
}

// TotalCount 返回所有层级的实体总数
func (em *EntityManager) TotalCount() int {
	em.mu.RLock()
	defer em.mu.RUnlock()

	total := 0
	for _, layerMap := range em.entitiesByLayer {
		total += len(*layerMap)
	}
	return total
}

// ClearLayer 清空指定层级的所有实体
func (em *EntityManager) ClearLayer(layer bmap.LayerNumber) {
	em.mu.Lock()
	defer em.mu.Unlock()

	delete(em.entitiesByLayer, layer)
}

// Clear 清空所有层级的实体
func (em *EntityManager) Clear() {
	em.mu.Lock()
	defer em.mu.Unlock()

	em.entitiesByLayer = make(map[bmap.LayerNumber]*map[bmap.EntityID]*bmap.Entity)
}

// HasEntity 检查指定层级是否存在实体
func (em *EntityManager) HasEntity(layer bmap.LayerNumber, entityID bmap.EntityID) bool {
	em.mu.RLock()
	defer em.mu.RUnlock()

	if layerMap, exists := em.entitiesByLayer[layer]; exists {
		_, found := (*layerMap)[entityID]
		return found
	}
	return false
}

// GetLayers 获取所有存在的层级
func (em *EntityManager) GetLayers() []bmap.LayerNumber {
	em.mu.RLock()
	defer em.mu.RUnlock()

	layers := make([]bmap.LayerNumber, 0, len(em.entitiesByLayer))
	for layer := range em.entitiesByLayer {
		layers = append(layers, layer)
	}
	return layers
}
