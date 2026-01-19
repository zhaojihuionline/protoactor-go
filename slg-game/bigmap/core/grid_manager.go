package core

import (
	"sync"

	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

// GridManager 网格管理器，负责管理游戏中的所有网格
type GridManager struct {
	grids map[int32]*bmap.Grid
	mu    sync.RWMutex
}

// NewGridManager 创建新的网格管理器
func NewGridManager() *GridManager {
	return &GridManager{
		grids: make(map[int32]*bmap.Grid),
	}
}

// AddGrid 添加网格到管理器
func (gm *GridManager) AddGrid(grid *bmap.Grid) error {
	if grid == nil {
		return nil // 忽略空网格
	}

	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.grids[grid.ID] = grid
	return nil
}

// RemoveGrid 从管理器中移除网格
func (gm *GridManager) RemoveGrid(gridID int32) {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	delete(gm.grids, gridID)
}

// GetGrid 根据ID获取网格
func (gm *GridManager) GetGrid(gridID int32) (*bmap.Grid, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	grid, exists := gm.grids[gridID]
	return grid, exists
}

// UpdateGrid 更新网格信息
func (gm *GridManager) UpdateGrid(grid *bmap.Grid) error {
	if grid == nil {
		return nil
	}

	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.grids[grid.ID] = grid
	return nil
}

// GetGridByPosition 根据位置获取网格
func (gm *GridManager) GetGridByPosition(x, y int32) (*bmap.Grid, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	for _, grid := range gm.grids {
		if grid.Position != nil && grid.Position.X == x && grid.Position.Y == y {
			return grid, true
		}
	}
	return nil, false
}

// GetGridsInArea 获取指定矩形区域内的所有网格
func (gm *GridManager) GetGridsInArea(minX, minY, maxX, maxY int32) []*bmap.Grid {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	var grids []*bmap.Grid
	for _, grid := range gm.grids {
		if grid.Position != nil {
			if grid.Position.X >= minX && grid.Position.X < maxX &&
				grid.Position.Y >= minY && grid.Position.Y < maxY {
				grids = append(grids, grid)
			}
		}
	}
	return grids
}

// GetGridsByEntity 获取指定实体的所有网格
func (gm *GridManager) GetGridsByEntity(entityID bmap.EntityID) []*bmap.Grid {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	var grids []*bmap.Grid
	for _, grid := range gm.grids {
		if grid.EntityID == entityID {
			grids = append(grids, grid)
		}
	}
	return grids
}

// GetAllGrids 获取所有网格
func (gm *GridManager) GetAllGrids() []*bmap.Grid {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	grids := make([]*bmap.Grid, 0, len(gm.grids))
	for _, grid := range gm.grids {
		grids = append(grids, grid)
	}
	return grids
}

// Count 返回网格总数
func (gm *GridManager) Count() int {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	return len(gm.grids)
}

// HasGrid 检查网格是否存在
func (gm *GridManager) HasGrid(gridID int32) bool {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	_, exists := gm.grids[gridID]
	return exists
}

// UpdateGridVersion 更新网格版本号
func (gm *GridManager) UpdateGridVersion(gridID int32) bool {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if grid, exists := gm.grids[gridID]; exists {
		grid.Version++
		return true
	}
	return false
}

// GetGridVersion 获取网格版本号
func (gm *GridManager) GetGridVersion(gridID int32) (int64, bool) {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	if grid, exists := gm.grids[gridID]; exists {
		return grid.Version, true
	}
	return 0, false
}

// Clear 清空所有网格
func (gm *GridManager) Clear() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	gm.grids = make(map[int32]*bmap.Grid)
}

// GetGridsWithVersionGreaterThan 获取版本号大于指定值的网格
func (gm *GridManager) GetGridsWithVersionGreaterThan(minVersion int64) []*bmap.Grid {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	var grids []*bmap.Grid
	for _, grid := range gm.grids {
		if grid.Version > minVersion {
			grids = append(grids, grid)
		}
	}
	return grids
}

// GetEmptyGrids 获取没有关联实体的空网格
func (gm *GridManager) GetEmptyGrids() []*bmap.Grid {
	gm.mu.RLock()
	defer gm.mu.RUnlock()

	var emptyGrids []*bmap.Grid
	for _, grid := range gm.grids {
		if grid.EntityID == 0 { // 假设 EntityID 为 0 表示空网格
			emptyGrids = append(emptyGrids, grid)
		}
	}
	return emptyGrids
}

// UpdateGridEntity 关联网格到实体
func (gm *GridManager) UpdateGridEntity(gridID int32, entityID bmap.EntityID) bool {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if grid, exists := gm.grids[gridID]; exists {
		grid.EntityID = entityID
		grid.Version++
		return true
	}
	return false
}

// RemoveGridEntity 移除网格的实体关联
func (gm *GridManager) RemoveGridEntity(gridID int32) bool {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if grid, exists := gm.grids[gridID]; exists {
		grid.EntityID = 0 // 重置为无实体关联
		grid.Version++
		return true
	}
	return false
}
