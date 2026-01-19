package core

import (
	"testing"

	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

func TestGridManager(t *testing.T) {
	gm := NewGridManager()

	// 测试添加网格
	grid1 := &bmap.Grid{
		ID:       1,
		Position: &bmap.Position{X: 10, Y: 20},
		Version:  1,
		EntityID: 100,
	}

	grid2 := &bmap.Grid{
		ID:       2,
		Position: &bmap.Position{X: 30, Y: 40},
		Version:  1,
		EntityID: 200,
	}

	grid3 := &bmap.Grid{
		ID:       3,
		Position: &bmap.Position{X: 50, Y: 60},
		Version:  1,
		EntityID: 0, // 空网格
	}

	err := gm.AddGrid(grid1)
	if err != nil {
		t.Errorf("AddGrid failed: %v", err)
	}

	err = gm.AddGrid(grid2)
	if err != nil {
		t.Errorf("AddGrid failed: %v", err)
	}

	err = gm.AddGrid(grid3)
	if err != nil {
		t.Errorf("AddGrid failed: %v", err)
	}

	// 测试计数
	if gm.Count() != 3 {
		t.Errorf("Expected 3 grids, got %d", gm.Count())
	}

	// 测试获取网格
	retrieved, exists := gm.GetGrid(1)
	if !exists {
		t.Error("Grid 1 should exist")
	}
	if retrieved.ID != 1 {
		t.Errorf("Expected grid ID 1, got %d", retrieved.ID)
	}

	// 测试按位置获取
	gridByPos, exists := gm.GetGridByPosition(30, 40)
	if !exists {
		t.Error("Grid at position (30,40) should exist")
	}
	if gridByPos.ID != 2 {
		t.Errorf("Expected grid ID 2, got %d", gridByPos.ID)
	}

	// 测试区域查询
	gridsInArea := gm.GetGridsInArea(5, 15, 35, 45)
	if len(gridsInArea) != 2 {
		t.Errorf("Expected 2 grids in area, got %d", len(gridsInArea))
	}

	// 测试按实体查询
	gridsByEntity := gm.GetGridsByEntity(100)
	if len(gridsByEntity) != 1 {
		t.Errorf("Expected 1 grid for entity 100, got %d", len(gridsByEntity))
	}

	// 测试空网格查询
	emptyGrids := gm.GetEmptyGrids()
	if len(emptyGrids) != 1 {
		t.Errorf("Expected 1 empty grid, got %d", len(emptyGrids))
	}

	// 测试版本管理
	version, exists := gm.GetGridVersion(1)
	if !exists || version != 1 {
		t.Errorf("Expected version 1, got %d", version)
	}

	// 更新版本
	if !gm.UpdateGridVersion(1) {
		t.Error("UpdateGridVersion should succeed")
	}

	newVersion, _ := gm.GetGridVersion(1)
	if newVersion != 2 {
		t.Errorf("Expected version 2 after update, got %d", newVersion)
	}

	// 测试实体关联更新
	if !gm.UpdateGridEntity(3, 300) {
		t.Error("UpdateGridEntity should succeed")
	}

	grid3Updated, _ := gm.GetGrid(3)
	if grid3Updated.EntityID != 300 {
		t.Errorf("Expected entity ID 300, got %d", grid3Updated.EntityID)
	}
	if grid3Updated.Version != 2 {
		t.Errorf("Expected version 2 after entity update, got %d", grid3Updated.Version)
	}

	// 测试移除实体关联
	if !gm.RemoveGridEntity(3) {
		t.Error("RemoveGridEntity should succeed")
	}

	grid3Removed, _ := gm.GetGrid(3)
	if grid3Removed.EntityID != 0 {
		t.Errorf("Expected entity ID 0 after removal, got %d", grid3Removed.EntityID)
	}

	// 测试移除网格
	gm.RemoveGrid(2)
	if gm.Count() != 2 {
		t.Errorf("Expected 2 grids after removal, got %d", gm.Count())
	}

	_, exists = gm.GetGrid(2)
	if exists {
		t.Error("Grid 2 should not exist after removal")
	}

	// 测试清空
	gm.Clear()
	if gm.Count() != 0 {
		t.Errorf("Expected 0 grids after clear, got %d", gm.Count())
	}
}

func TestGridManager_EdgeCases(t *testing.T) {
	gm := NewGridManager()

	// 测试添加 nil 网格
	err := gm.AddGrid(nil)
	if err != nil {
		t.Errorf("AddGrid with nil should not error: %v", err)
	}

	// 测试不存在的网格
	_, exists := gm.GetGrid(999)
	if exists {
		t.Error("Non-existent grid should not be found")
	}

	// 测试版本更新不存在的网格
	if gm.UpdateGridVersion(999) {
		t.Error("UpdateGridVersion on non-existent grid should fail")
	}

	// 测试实体更新不存在的网格
	if gm.UpdateGridEntity(999, 123) {
		t.Error("UpdateGridEntity on non-existent grid should fail")
	}
}