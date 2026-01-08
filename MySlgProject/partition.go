package main

import (
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// Partition 地图分区
type Partition struct {
	ID           string
	Bounds       Rectangle
	Size         int
	GridStates   map[Point]*GridState
	Entities     map[string]Entity
	Subscribers  map[string]*AOISubscriber // 玩家ID -> AOI订阅者
	StateVersion int64
	mutex        sync.RWMutex
	config       *Config
	system       *actor.ActorSystem
}

// AOISubscriber AOI订阅者
type AOISubscriber struct {
	PlayerID   string
	AOIManager *actor.PID
	ViewRect   Rectangle
	LastUpdate time.Time
}

// NewPartition 创建新分区
func NewPartition(id string, bounds Rectangle, config *Config, system *actor.ActorSystem) *Partition {
	return &Partition{
		ID:           id,
		Bounds:       bounds,
		Size:         bounds.Width, // 假设正方形分区
		GridStates:   make(map[Point]*GridState),
		Entities:     make(map[string]Entity),
		Subscribers:  make(map[string]*AOISubscriber),
		StateVersion: 1,
		config:       config,
		system:       system,
	}
}

// Initialize 初始化分区数据
func (p *Partition) Initialize() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// 初始化所有格子的基础状态
	for x := p.Bounds.X; x < p.Bounds.X+p.Bounds.Width; x++ {
		for y := p.Bounds.Y; y < p.Bounds.Y+p.Bounds.Height; y++ {
			pos := Point{X: x, Y: y}
			p.GridStates[pos] = &GridState{
				Position:   pos,
				Terrain:    TerrainGrass, // 默认草地
				Resource:   nil,
				Building:   nil,
				Occupants:  make([]*Entity, 0),
				LastUpdate: time.Now(),
				Version:    1,
			}
		}
	}
}

// AddSubscriber 添加AOI订阅者
func (p *Partition) AddSubscriber(playerID string, aoiManager *actor.PID, viewRect Rectangle) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.Subscribers[playerID] = &AOISubscriber{
		PlayerID:   playerID,
		AOIManager: aoiManager,
		ViewRect:   viewRect,
		LastUpdate: time.Now(),
	}
}

// RemoveSubscriber 移除AOI订阅者
func (p *Partition) RemoveSubscriber(playerID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.Subscribers, playerID)
}

// UpdateSubscriberView 更新订阅者视野
func (p *Partition) UpdateSubscriberView(playerID string, viewRect Rectangle) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if sub, exists := p.Subscribers[playerID]; exists {
		sub.ViewRect = viewRect
		sub.LastUpdate = time.Now()
	}
}

// GetGridState 获取格子状态
func (p *Partition) GetGridState(pos Point) *GridState {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.GridStates[pos]
}

// UpdateGridState 更新格子状态
func (p *Partition) UpdateGridState(pos Point, newState *GridState) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if oldState, exists := p.GridStates[pos]; exists {
		// 创建状态变更记录
		change := &StateChange{
			GridPos:    pos,
			ChangeType: ChangeTerrain, // 这里可以根据具体变更确定类型
			OldState:   oldState,
			NewState:   newState,
			Timestamp:  time.Now(),
			Version:    p.StateVersion + 1,
		}

		// 更新状态
		newState.LastUpdate = time.Now()
		newState.Version = change.Version
		p.GridStates[pos] = newState
		p.StateVersion = change.Version

		// 通知相关订阅者
		p.notifySubscribers(change)
	}
}

// AddEntity 添加实体到分区
func (p *Partition) AddEntity(entity Entity) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.Entities[entity.GetID()] = entity
}

// RemoveEntity 从分区移除实体
func (p *Partition) RemoveEntity(entityID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	delete(p.Entities, entityID)
}

// GetEntitiesInRange 获取范围内的实体
func (p *Partition) GetEntitiesInRange(center Point, radius int) []Entity {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]Entity, 0)
	for _, entity := range p.Entities {
		pos := entity.GetPosition()
		if center.Distance(pos) <= float64(radius*radius) {
			result = append(result, entity)
		}
	}
	return result
}

// GetEntitiesInRect 获取矩形区域内的实体
func (p *Partition) GetEntitiesInRect(rect Rectangle) []Entity {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := make([]Entity, 0)
	for _, entity := range p.Entities {
		pos := entity.GetPosition()
		if rect.Contains(pos) {
			result = append(result, entity)
		}
	}
	return result
}

// GetSubscriberCount 获取订阅者数量
func (p *Partition) GetSubscriberCount() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return len(p.Subscribers)
}

// GetEntityCount 获取实体数量
func (p *Partition) GetEntityCount() int {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return len(p.Entities)
}

// GetMetrics 获取分区指标
func (p *Partition) GetMetrics() *PartitionMetrics {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return &PartitionMetrics{
		PartitionID:     p.ID,
		PlayerCount:     len(p.Subscribers),
		EntityCount:     len(p.Entities),
		MessageRate:     0, // 需要额外的消息计数器
		CpuUsage:        0, // 需要性能监控
		MemoryUsage:     0, // 需要内存监控
		AvgResponseTime: 0, // 需要响应时间监控
		LastUpdate:      time.Now(),
	}
}

// notifySubscribers 通知相关订阅者状态变更
func (p *Partition) notifySubscribers(change *StateChange) {
	for playerID, subscriber := range p.Subscribers {
		// 检查变更是否在订阅者的视野范围内
		if subscriber.ViewRect.Contains(change.GridPos) {
			// 发送状态变更到AOI管理器
			message := &AOIStateChange{
				PartitionID: p.ID,
				Change:      change,
				PlayerID:    playerID,
			}

			p.system.Root.Send(subscriber.AOIManager, message)
		}
	}
}

// AOIStateChange AOI状态变更消息
type AOIStateChange struct {
	PartitionID string
	Change      *StateChange
	PlayerID    string
}

// Split 划分分区（用于动态分区调整）
func (p *Partition) Split(newSize int) []*Partition {
	// 简单的四等分划分
	halfSize := p.Size / 2
	if halfSize < p.config.Partition.MinSize {
		return nil // 无法再分割
	}

	partitions := make([]*Partition, 4)

	// 创建四个子分区
	partitions[0] = NewPartition(p.ID+"_00", Rectangle{
		X: p.Bounds.X, Y: p.Bounds.Y,
		Width: halfSize, Height: halfSize,
	}, p.config, p.system)

	partitions[1] = NewPartition(p.ID+"_01", Rectangle{
		X: p.Bounds.X + halfSize, Y: p.Bounds.Y,
		Width: halfSize, Height: halfSize,
	}, p.config, p.system)

	partitions[2] = NewPartition(p.ID+"_10", Rectangle{
		X: p.Bounds.X, Y: p.Bounds.Y + halfSize,
		Width: halfSize, Height: halfSize,
	}, p.config, p.system)

	partitions[3] = NewPartition(p.ID+"_11", Rectangle{
		X: p.Bounds.X + halfSize, Y: p.Bounds.Y + halfSize,
		Width: halfSize, Height: halfSize,
	}, p.config, p.system)

	// 重新分配实体和状态到新分区
	p.redistributeEntities(partitions)
	p.redistributeGridStates(partitions)

	return partitions
}

// redistributeEntities 重新分配实体
func (p *Partition) redistributeEntities(newPartitions []*Partition) {
	for entityID, entity := range p.Entities {
		pos := entity.GetPosition()
		targetPartition := p.findTargetPartition(pos, newPartitions)
		if targetPartition != nil {
			targetPartition.AddEntity(entity)
		}
		delete(p.Entities, entityID)
	}
}

// redistributeGridStates 重新分配格子状态
func (p *Partition) redistributeGridStates(newPartitions []*Partition) {
	for pos, state := range p.GridStates {
		targetPartition := p.findTargetPartition(pos, newPartitions)
		if targetPartition != nil {
			targetPartition.GridStates[pos] = state
		}
		delete(p.GridStates, pos)
	}
}

// findTargetPartition 查找目标分区
func (p *Partition) findTargetPartition(pos Point, partitions []*Partition) *Partition {
	for _, partition := range partitions {
		if partition.Bounds.Contains(pos) {
			return partition
		}
	}
	return nil
}
