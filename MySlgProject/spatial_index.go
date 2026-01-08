package main

import (
	"fmt"
	"sync"
)

// SpatialIndex 空间索引 - 使用四叉树进行空间分区
type SpatialIndex struct {
	root         *QuadNode
	regionSize   int
	mutex        sync.RWMutex
	config       *Config
}

// QuadNode 四叉树节点
type QuadNode struct {
	bounds     Rectangle
	children   [4]*QuadNode // 四个子节点：左上、右上、左下、右下
	entities   map[string]Entity
	isLeaf     bool
	level      int
	maxLevel   int
	maxEntities int
}

// NewSpatialIndex 创建空间索引
func NewSpatialIndex(config *Config) *SpatialIndex {
	rootBounds := Rectangle{
		X:      0,
		Y:      0,
		Width:  config.World.Width,
		Height: config.World.Height,
	}

	root := &QuadNode{
		bounds:      rootBounds,
		entities:    make(map[string]Entity),
		isLeaf:      true,
		level:       0,
		maxLevel:    6, // 最大深度
		maxEntities: 16, // 每个节点最大实体数
	}

	return &SpatialIndex{
		root:       root,
		regionSize: config.Partition.DefaultSize,
		config:     config,
	}
}

// GetPartitionAt 获取坐标所在的区域ID
func (si *SpatialIndex) GetPartitionAt(point Point) string {
	regionX := point.X / si.regionSize
	regionY := point.Y / si.regionSize
	return fmt.Sprintf("region_%d_%d", regionX, regionY)
}

// GetPartitionBounds 获取区域边界
func (si *SpatialIndex) GetPartitionBounds(partitionID string) Rectangle {
	var regionX, regionY int
	fmt.Sscanf(partitionID, "region_%d_%d", &regionX, &regionY)

	return Rectangle{
		X:      regionX * si.regionSize,
		Y:      regionY * si.regionSize,
		Width:  si.regionSize,
		Height: si.regionSize,
	}
}

// GetPartitionsInRange 获取范围内的所有分区
func (si *SpatialIndex) GetPartitionsInRange(center Point, radius int) []string {
	minX := max(0, center.X-radius)
	maxX := min(si.config.World.Width-1, center.X+radius)
	minY := max(0, center.Y-radius)
	maxY := min(si.config.World.Height-1, center.Y+radius)

	partitions := make(map[string]bool)

	for x := minX; x <= maxX; x += si.regionSize {
		for y := minY; y <= maxY; y += si.regionSize {
			partitionID := si.GetPartitionAt(Point{x, y})
			partitions[partitionID] = true
		}
	}

	result := make([]string, 0, len(partitions))
	for partitionID := range partitions {
		result = append(result, partitionID)
	}

	return result
}

// InsertEntity 插入实体到空间索引
func (si *SpatialIndex) InsertEntity(entity Entity) {
	si.mutex.Lock()
	defer si.mutex.Unlock()

	pos := entity.GetPosition()
	node := si.findLeafNode(si.root, pos)
	node.entities[entity.GetID()] = entity

	// 检查是否需要分裂
	if len(node.entities) > node.maxEntities && node.level < node.maxLevel {
		si.splitNode(node)
	}
}

// RemoveEntity 从空间索引移除实体
func (si *SpatialIndex) RemoveEntity(entityID string) {
	si.mutex.Lock()
	defer si.mutex.Unlock()

	// 遍历所有节点查找实体
	si.removeEntityFromNode(si.root, entityID)
}

// UpdateEntityPosition 更新实体位置
func (si *SpatialIndex) UpdateEntityPosition(entityID string, newPos Point) {
	si.mutex.Lock()
	defer si.mutex.Unlock()

	// 先移除旧位置
	oldEntity := si.removeEntityFromNode(si.root, entityID)
	if oldEntity == nil {
		return
	}

	// 设置新位置并插入
	oldEntity.SetPosition(newPos)
	pos := newPos
	node := si.findLeafNode(si.root, pos)
	node.entities[entityID] = oldEntity

	// 检查是否需要分裂
	if len(node.entities) > node.maxEntities && node.level < node.maxLevel {
		si.splitNode(node)
	}
}

// QueryEntitiesInRange 查询范围内的实体
func (si *SpatialIndex) QueryEntitiesInRange(center Point, radius int) []Entity {
	si.mutex.RLock()
	defer si.mutex.RUnlock()

	result := make([]Entity, 0)
	queryRect := Rectangle{
		X:      center.X - radius,
		Y:      center.Y - radius,
		Width:  radius * 2,
		Height: radius * 2,
	}

	si.queryEntitiesInNode(si.root, queryRect, &result)
	return result
}

// QueryEntitiesInPartition 查询分区内的所有实体
func (si *SpatialIndex) QueryEntitiesInPartition(partitionID string) []Entity {
	bounds := si.GetPartitionBounds(partitionID)
	return si.QueryEntitiesInRect(bounds)
}

// QueryEntitiesInRect 查询矩形区域内的实体
func (si *SpatialIndex) QueryEntitiesInRect(rect Rectangle) []Entity {
	si.mutex.RLock()
	defer si.mutex.RUnlock()

	result := make([]Entity, 0)
	si.queryEntitiesInNode(si.root, rect, &result)
	return result
}

// findLeafNode 查找叶子节点
func (si *SpatialIndex) findLeafNode(node *QuadNode, pos Point) *QuadNode {
	if node.isLeaf {
		return node
	}

	// 确定子节点索引
	childIndex := 0
	if pos.X >= node.bounds.X+node.bounds.Width/2 {
		childIndex |= 1 // 右边
	}
	if pos.Y >= node.bounds.Y+node.bounds.Height/2 {
		childIndex |= 2 // 下边
	}

	return si.findLeafNode(node.children[childIndex], pos)
}

// splitNode 分裂节点
func (si *SpatialIndex) splitNode(node *QuadNode) {
	if !node.isLeaf {
		return
	}

	halfWidth := node.bounds.Width / 2
	halfHeight := node.bounds.Height / 2

	// 创建四个子节点
	node.children[0] = &QuadNode{ // 左上
		bounds: Rectangle{
			X:      node.bounds.X,
			Y:      node.bounds.Y,
			Width:  halfWidth,
			Height: halfHeight,
		},
		entities:    make(map[string]Entity),
		isLeaf:      true,
		level:       node.level + 1,
		maxLevel:    node.maxLevel,
		maxEntities: node.maxEntities,
	}

	node.children[1] = &QuadNode{ // 右上
		bounds: Rectangle{
			X:      node.bounds.X + halfWidth,
			Y:      node.bounds.Y,
			Width:  halfWidth,
			Height: halfHeight,
		},
		entities:    make(map[string]Entity),
		isLeaf:      true,
		level:       node.level + 1,
		maxLevel:    node.maxLevel,
		maxEntities: node.maxEntities,
	}

	node.children[2] = &QuadNode{ // 左下
		bounds: Rectangle{
			X:      node.bounds.X,
			Y:      node.bounds.Y + halfHeight,
			Width:  halfWidth,
			Height: halfHeight,
		},
		entities:    make(map[string]Entity),
		isLeaf:      true,
		level:       node.level + 1,
		maxLevel:    node.maxLevel,
		maxEntities: node.maxEntities,
	}

	node.children[3] = &QuadNode{ // 右下
		bounds: Rectangle{
			X:      node.bounds.X + halfWidth,
			Y:      node.bounds.Y + halfHeight,
			Width:  halfWidth,
			Height: halfHeight,
		},
		entities:    make(map[string]Entity),
		isLeaf:      true,
		level:       node.level + 1,
		maxLevel:    node.maxLevel,
		maxEntities: node.maxEntities,
	}

	// 重新分配实体到子节点
	for entityID, entity := range node.entities {
		pos := entity.GetPosition()
		childIndex := 0
		if pos.X >= node.bounds.X+halfWidth {
			childIndex |= 1
		}
		if pos.Y >= node.bounds.Y+halfHeight {
			childIndex |= 2
		}
		node.children[childIndex].entities[entityID] = entity
	}

	// 清空当前节点实体
	node.entities = nil
	node.isLeaf = false
}

// removeEntityFromNode 从节点及其子节点中移除实体
func (si *SpatialIndex) removeEntityFromNode(node *QuadNode, entityID string) Entity {
	if node.isLeaf {
		if entity, exists := node.entities[entityID]; exists {
			delete(node.entities, entityID)
			return entity
		}
		return nil
	}

	// 递归查找子节点
	for _, child := range node.children {
		if entity := si.removeEntityFromNode(child, entityID); entity != nil {
			return entity
		}
	}

	return nil
}

// queryEntitiesInNode 查询节点内的实体
func (si *SpatialIndex) queryEntitiesInNode(node *QuadNode, queryRect Rectangle, result *[]Entity) {
	if !node.bounds.Intersects(queryRect) {
		return
	}

	if node.isLeaf {
		for _, entity := range node.entities {
			pos := entity.GetPosition()
			if queryRect.Contains(pos) {
				*result = append(*result, entity)
			}
		}
		return
	}

	// 递归查询子节点
	for _, child := range node.children {
		si.queryEntitiesInNode(child, queryRect, result)
	}
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
