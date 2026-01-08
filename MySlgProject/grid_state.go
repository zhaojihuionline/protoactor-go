package main

import (
	"sync"
	"time"
)

// GridStateManager 格子状态管理器
type GridStateManager struct {
	states       map[Point]*GridState
	mutex        sync.RWMutex
	changeQueue  chan *StateChange
	config       *Config
	system       *actor.ActorSystem
}

// NewGridStateManager 创建格子状态管理器
func NewGridStateManager(config *Config, system *actor.ActorSystem) *GridStateManager {
	return &GridStateManager{
		states:      make(map[Point]*GridState),
		changeQueue: make(chan *StateChange, 1000),
		config:      config,
		system:      system,
	}
}

// Initialize 初始化地图格子状态
func (gsm *GridStateManager) Initialize() {
	gsm.mutex.Lock()
	defer gsm.mutex.Unlock()

	// 初始化整个地图的格子状态
	for x := 0; x < gsm.config.World.Width; x++ {
		for y := 0; y < gsm.config.World.Height; y++ {
			pos := Point{X: x, Y: y}
			gsm.states[pos] = &GridState{
				Position:   pos,
				Terrain:    gsm.generateTerrain(pos),
				Resource:   gsm.generateResource(pos),
				Building:   nil,
				Occupants:  make([]*Entity, 0),
				LastUpdate: time.Now(),
				Version:    1,
			}
		}
	}
}

// GetState 获取格子状态
func (gsm *GridStateManager) GetState(pos Point) *GridState {
	gsm.mutex.RLock()
	defer gsm.mutex.RUnlock()

	return gsm.states[pos]
}

// UpdateState 更新格子状态
func (gsm *GridStateManager) UpdateState(pos Point, updater func(*GridState) *GridState) {
	gsm.mutex.Lock()
	defer gsm.mutex.Unlock()

	if state, exists := gsm.states[pos]; exists {
		oldState := *state // 复制旧状态
		newState := updater(state)

		if newState != nil {
			newState.LastUpdate = time.Now()
			newState.Version = oldState.Version + 1
			gsm.states[pos] = newState

			// 发送状态变更到队列
			change := &StateChange{
				GridPos:    pos,
				ChangeType: ChangeTerrain, // 这里可以根据具体变更确定类型
				OldState:   &oldState,
				NewState:   newState,
				Timestamp:  time.Now(),
				Version:    newState.Version,
			}

			select {
			case gsm.changeQueue <- change:
				// 成功加入队列
			default:
				// 队列满，异步处理
				go func() {
					gsm.changeQueue <- change
				}()
			}
		}
	}
}

// BatchUpdate 批量更新状态
func (gsm *GridStateManager) BatchUpdate(updates map[Point]func(*GridState) *GridState) {
	gsm.mutex.Lock()
	defer gsm.mutex.Unlock()

	changes := make([]*StateChange, 0, len(updates))

	for pos, updater := range updates {
		if state, exists := gsm.states[pos]; exists {
			oldState := *state
			newState := updater(state)

			if newState != nil {
				newState.LastUpdate = time.Now()
				newState.Version = oldState.Version + 1
				gsm.states[pos] = newState

				change := &StateChange{
					GridPos:    pos,
					ChangeType: ChangeTerrain,
					OldState:   &oldState,
					NewState:   newState,
					Timestamp:  time.Now(),
					Version:    newState.Version,
				}
				changes = append(changes, change)
			}
		}
	}

	// 批量发送变更
	for _, change := range changes {
		select {
		case gsm.changeQueue <- change:
		default:
			go func(ch *StateChange) {
				gsm.changeQueue <- ch
			}(change)
		}
	}
}

// GetStatesInRange 获取范围内的格子状态
func (gsm *GridStateManager) GetStatesInRange(center Point, radius int) []*GridState {
	gsm.mutex.RLock()
	defer gsm.mutex.RUnlock()

	result := make([]*GridState, 0)
	minX := max(0, center.X-radius)
	maxX := min(gsm.config.World.Width-1, center.X+radius)
	minY := max(0, center.Y-radius)
	maxY := min(gsm.config.World.Height-1, center.Y+radius)

	for x := minX; x <= maxX; x++ {
		for y := minY; y <= maxY; y++ {
			pos := Point{X: x, Y: y}
			if state, exists := gsm.states[pos]; exists {
				distance := center.Distance(pos)
				if distance <= float64(radius*radius) {
					result = append(result, state)
				}
			}
		}
	}

	return result
}

// GetChangeQueue 获取状态变更队列
func (gsm *GridStateManager) GetChangeQueue() <-chan *StateChange {
	return gsm.changeQueue
}

// GetStats 获取统计信息
func (gsm *GridStateManager) GetStats() map[string]interface{} {
	gsm.mutex.RLock()
	defer gsm.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_grids"] = len(gsm.states)
	stats["queue_size"] = len(gsm.changeQueue)

	// 计算地形分布
	terrainCount := make(map[TerrainType]int)
	for _, state := range gsm.states {
		terrainCount[state.Terrain]++
	}
	stats["terrain_distribution"] = terrainCount

	// 计算建筑数量
	buildingCount := 0
	for _, state := range gsm.states {
		if state.Building != nil {
			buildingCount++
		}
	}
	stats["building_count"] = buildingCount

	return stats
}

// generateTerrain 生成地形
func (gsm *GridStateManager) generateTerrain(pos Point) TerrainType {
	// 简单的地形生成算法
	// 实际游戏中可以使用Perlin噪声等更复杂的算法
	x, y := float64(pos.X), float64(pos.Y)

	// 山脉
	if x > 300 && x < 500 && y > 200 && y < 400 {
		return TerrainMountain
	}

	// 森林
	if x > 100 && x < 300 && y > 500 && y < 700 {
		return TerrainForest
	}

	// 水域
	if x > 600 && x < 800 && y > 100 && y < 300 {
		return TerrainWater
	}

	// 沙漠
	if x > 800 && x < 1000 && y > 600 && y < 800 {
		return TerrainDesert
	}

	// 默认草地
	return TerrainGrass
}

// generateResource 生成资源
func (gsm *GridStateManager) generateResource(pos Point) *Resource {
	// 简单的资源生成逻辑
	// 森林产生木材
	if gsm.states[pos].Terrain == TerrainForest {
		return &Resource{
			Type:       ResourceWood,
			Amount:     100,
			RegenRate:  5,
			LastHarvest: time.Now(),
		}
	}

	// 山脉产生石头
	if gsm.states[pos].Terrain == TerrainMountain {
		return &Resource{
			Type:       ResourceStone,
			Amount:     80,
			RegenRate:  3,
			LastHarvest: time.Now(),
		}
	}

	// 农田产生食物（在草地上）
	if gsm.states[pos].Terrain == TerrainGrass && pos.X%50 == 0 && pos.Y%50 == 0 {
		return &Resource{
			Type:       ResourceFood,
			Amount:     60,
			RegenRate:  8,
			LastHarvest: time.Now(),
		}
	}

	return nil
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
