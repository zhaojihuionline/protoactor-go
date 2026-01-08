package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// PartitionAdjuster 分区调整器 - 执行具体的分区调整操作
type PartitionAdjuster struct {
	config       *Config
	system       *actor.ActorSystem
	partitionMgr *PartitionManager
	spatialIndex *SpatialIndex
	activeOps    sync.Map // map[string]*AdjustmentOperation
	mutex        sync.Mutex
}

// AdjustmentOperation 调整操作
type AdjustmentOperation struct {
	Plan        *AdjustmentPlan
	StartTime   time.Time
	Status      OperationStatus
	Error       error
	SubTasks    []*MigrationTask
	CompletedAt *time.Time
}

// OperationStatus 操作状态
type OperationStatus int

const (
	StatusPending OperationStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusRolledBack
)

// NewPartitionAdjuster 创建分区调整器
func NewPartitionAdjuster(config *Config, system *actor.ActorSystem,
	partitionMgr *PartitionManager, spatialIndex *SpatialIndex) *PartitionAdjuster {

	return &PartitionAdjuster{
		config:       config,
		system:       system,
		partitionMgr: partitionMgr,
		spatialIndex: spatialIndex,
	}
}

// ExecuteAdjustment 执行调整
func (pa *PartitionAdjuster) ExecuteAdjustment(plan *AdjustmentPlan) error {
	pa.mutex.Lock()
	defer pa.mutex.Unlock()

	// 检查是否已有正在进行的操作
	if _, exists := pa.activeOps.Load(plan.PartitionID); exists {
		return fmt.Errorf("partition %s is already being adjusted", plan.PartitionID)
	}

	// 创建调整操作
	operation := &AdjustmentOperation{
		Plan:      plan,
		StartTime: time.Now(),
		Status:    StatusPending,
		SubTasks:  make([]*MigrationTask, 0),
	}

	pa.activeOps.Store(plan.PartitionID, operation)

	// 执行调整
	var err error
	switch plan.Action {
	case ActionSplit:
		err = pa.executeSplit(plan)
	case ActionMerge:
		err = pa.executeMerge(plan)
	case ActionResize:
		err = pa.executeResize(plan)
	default:
		err = fmt.Errorf("unsupported adjustment action: %v", plan.Action)
	}

	// 更新操作状态
	operation.Status = StatusCompleted
	if err != nil {
		operation.Status = StatusFailed
		operation.Error = err
	} else {
		now := time.Now()
		operation.CompletedAt = &now
	}

	return err
}

// executeSplit 执行分割
func (pa *PartitionAdjuster) executeSplit(plan *AdjustmentPlan) error {
	partitionID := plan.PartitionID

	// 获取原分区
	originalPartition, exists := pa.partitionMgr.GetPartition(partitionID)
	if !exists {
		return fmt.Errorf("partition %s not found", partitionID)
	}

	// 执行分割
	newPartitions := originalPartition.Split(plan.NewSize)
	if newPartitions == nil {
		return fmt.Errorf("failed to split partition %s", partitionID)
	}

	// 初始化新分区
	for _, newPartition := range newPartitions {
		newPartition.Initialize()
		pa.partitionMgr.AddPartition(newPartition)
	}

	// 更新空间索引
	pa.updateSpatialIndexForSplit(partitionID, newPartitions)

	// 迁移订阅者
	pa.migrateSubscribersForSplit(originalPartition, newPartitions)

	// 清理原分区
	pa.partitionMgr.RemovePartition(partitionID)

	return nil
}

// executeMerge 执行合并
func (pa *PartitionAdjuster) executeMerge(plan *AdjustmentPlan) error {
	mainPartitionID := plan.PartitionID
	affectedPartitionIDs := plan.AffectedPartitions

	// 获取所有参与合并的分区
	partitions := make([]*Partition, 0, len(affectedPartitionIDs)+1)

	mainPartition, exists := pa.partitionMgr.GetPartition(mainPartitionID)
	if !exists {
		return fmt.Errorf("main partition %s not found", mainPartitionID)
	}
	partitions = append(partitions, mainPartition)

	for _, partitionID := range affectedPartitionIDs {
		partition, exists := pa.partitionMgr.GetPartition(partitionID)
		if !exists {
			return fmt.Errorf("affected partition %s not found", partitionID)
		}
		partitions = append(partitions, partition)
	}

	// 创建合并后的分区
	mergedPartition := pa.createMergedPartition(partitions)
	mergedPartition.Initialize()

	// 迁移所有数据
	if err := pa.migrateDataToMergedPartition(partitions, mergedPartition); err != nil {
		return fmt.Errorf("failed to migrate data: %w", err)
	}

	// 添加新分区
	pa.partitionMgr.AddPartition(mergedPartition)

	// 更新空间索引
	pa.updateSpatialIndexForMerge(partitions, mergedPartition)

	// 迁移订阅者
	pa.migrateSubscribersForMerge(partitions, mergedPartition)

	// 清理原分区
	for _, partition := range partitions {
		pa.partitionMgr.RemovePartition(partition.ID)
	}

	return nil
}

// executeResize 执行调整大小
func (pa *PartitionAdjuster) executeResize(plan *AdjustmentPlan) error {
	// 调整大小通常不需要数据迁移，只是改变边界
	_, exists := pa.partitionMgr.GetPartition(plan.PartitionID)
	if !exists {
		return fmt.Errorf("partition %s not found", plan.PartitionID)
	}

	// 这里可以实现更复杂的调整大小逻辑
	// 目前暂时不支持
	return fmt.Errorf("resize operation not implemented")
}

// createMergedPartition 创建合并后的分区
func (pa *PartitionAdjuster) createMergedPartition(partitions []*Partition) *Partition {
	// 计算合并后的边界
	minX, minY := partitions[0].Bounds.X, partitions[0].Bounds.Y
	maxX, maxY := partitions[0].Bounds.X+partitions[0].Bounds.Width,
				  partitions[0].Bounds.Y+partitions[0].Bounds.Height

	for _, partition := range partitions[1:] {
		if partition.Bounds.X < minX {
			minX = partition.Bounds.X
		}
		if partition.Bounds.Y < minY {
			minY = partition.Bounds.Y
		}
		if partition.Bounds.X+partition.Bounds.Width > maxX {
			maxX = partition.Bounds.X + partition.Bounds.Width
		}
		if partition.Bounds.Y+partition.Bounds.Height > maxY {
			maxY = partition.Bounds.Y + partition.Bounds.Height
		}
	}

	mergedBounds := Rectangle{
		X:      minX,
		Y:      minY,
		Width:  maxX - minX,
		Height: maxY - minY,
	}

	// 使用第一个分区ID作为合并后的ID
	mergedID := partitions[0].ID + "_merged"

	return NewPartition(mergedID, mergedBounds, pa.config, pa.system)
}

// migrateDataToMergedPartition 迁移数据到合并分区
func (pa *PartitionAdjuster) migrateDataToMergedPartition(sources []*Partition, target *Partition) error {
	for _, source := range sources {
		// 迁移实体
		for _, entity := range source.Entities {
			target.AddEntity(entity)
		}

		// 迁移格子状态
		for pos, state := range source.GridStates {
			target.GridStates[pos] = state
		}
	}

	return nil
}

// updateSpatialIndexForSplit 更新空间索引（分割）
func (pa *PartitionAdjuster) updateSpatialIndexForSplit(oldPartitionID string, newPartitions []*Partition) {
	// 移除旧分区索引
	// 实际实现中需要更新空间索引

	// 添加新分区索引
	for _, partition := range newPartitions {
		// 更新空间索引
		pa.spatialIndex.InsertEntity(&BaseEntity{
			ID:       partition.ID,
			Position: Point{X: partition.Bounds.X, Y: partition.Bounds.Y},
			Type:     EntityBuilding, // 用作分区标记
		})
	}
}

// updateSpatialIndexForMerge 更新空间索引（合并）
func (pa *PartitionAdjuster) updateSpatialIndexForMerge(oldPartitions []*Partition, newPartition *Partition) {
	// 移除旧分区索引
	for _, partition := range oldPartitions {
		pa.spatialIndex.RemoveEntity(partition.ID)
	}

	// 添加新分区索引
	pa.spatialIndex.InsertEntity(&BaseEntity{
		ID:       newPartition.ID,
		Position: Point{X: newPartition.Bounds.X, Y: newPartition.Bounds.Y},
		Type:     EntityBuilding,
	})
}

// migrateSubscribersForSplit 迁移订阅者（分割）
func (pa *PartitionAdjuster) migrateSubscribersForSplit(originalPartition *Partition, newPartitions []*Partition) {
	// 通知所有订阅者分区已分割
	for playerID, subscriber := range originalPartition.Subscribers {
		// 确定订阅者应该迁移到哪个新分区
		newPartition := pa.findPartitionForSubscriber(subscriber, newPartitions)
		if newPartition != nil {
			// 取消对原分区的订阅
			originalPartition.RemoveSubscriber(playerID)

			// 添加到新分区
			newPartition.AddSubscriber(playerID, subscriber.AOIManager, subscriber.ViewRect)

			// 通知AOI管理器
			pa.notifyPartitionChange(playerID, originalPartition.ID, newPartition.ID)
		}
	}
}

// migrateSubscribersForMerge 迁移订阅者（合并）
func (pa *PartitionAdjuster) migrateSubscribersForMerge(oldPartitions []*Partition, newPartition *Partition) {
	// 收集所有订阅者
	allSubscribers := make(map[string]*PartitionSubscriber)

	for _, partition := range oldPartitions {
		for playerID, subscriber := range partition.Subscribers {
			allSubscribers[playerID] = subscriber
			partition.RemoveSubscriber(playerID)
		}
	}

	// 添加到新分区
	for playerID, subscriber := range allSubscribers {
		newPartition.AddSubscriber(playerID, subscriber.AOIManager, subscriber.ViewRect)

		// 通知AOI管理器
		oldIDs := make([]string, len(oldPartitions))
		for i, p := range oldPartitions {
			oldIDs[i] = p.ID
		}
		pa.notifyPartitionMerge(playerID, oldIDs, newPartition.ID)
	}
}

// findPartitionForSubscriber 查找订阅者所属的新分区
func (pa *PartitionAdjuster) findPartitionForSubscriber(subscriber *PartitionSubscriber, partitions []*Partition) *Partition {
	// 根据订阅者的视图中心点确定所属分区
	centerX := subscriber.ViewRect.X + subscriber.ViewRect.Width/2
	centerY := subscriber.ViewRect.Y + subscriber.ViewRect.Height/2
	centerPos := Point{X: centerX, Y: centerY}

	for _, partition := range partitions {
		if partition.Bounds.Contains(centerPos) {
			return partition
		}
	}

	// 如果没有找到，使用第一个分区
	return partitions[0]
}

// notifyPartitionChange 通知分区变化
func (pa *PartitionAdjuster) notifyPartitionChange(playerID, oldPartitionID, newPartitionID string) {
	// 发送分区变化通知到玩家的AOI管理器
	changeMsg := &PartitionChanged{
		PlayerID:       playerID,
		OldPartitionID: oldPartitionID,
		NewPartitionID: newPartitionID,
		ChangeType:     PartitionSplit,
	}

	// 这里需要获取玩家的AOI管理器PID
	// 暂时通过消息广播
	pa.system.EventStream.Publish(changeMsg)
}

// notifyPartitionMerge 通知分区合并
func (pa *PartitionAdjuster) notifyPartitionMerge(playerID string, oldPartitionIDs []string, newPartitionID string) {
	mergeMsg := &PartitionChanged{
		PlayerID:        playerID,
		OldPartitionIDs: oldPartitionIDs,
		NewPartitionID:  newPartitionID,
		ChangeType:      PartitionMerge,
	}

	pa.system.EventStream.Publish(mergeMsg)
}

// GetActiveOperations 获取活跃的操作
func (pa *PartitionAdjuster) GetActiveOperations() map[string]*AdjustmentOperation {
	operations := make(map[string]*AdjustmentOperation)

	pa.activeOps.Range(func(key, value interface{}) bool {
		partitionID := key.(string)
		operation := value.(*AdjustmentOperation)
		operations[partitionID] = operation
		return true
	})

	return operations
}

// CleanupCompletedOperations 清理已完成的操作
func (pa *PartitionAdjuster) CleanupCompletedOperations() {
	pa.activeOps.Range(func(key, value interface{}) bool {
		operation := value.(*AdjustmentOperation)
		if operation.Status == StatusCompleted || operation.Status == StatusFailed {
			// 保留5分钟的完成记录
			if time.Since(operation.StartTime) > 5*time.Minute {
				pa.activeOps.Delete(key)
			}
		}
		return true
	})
}

// PartitionChanged 分区变化消息
type PartitionChanged struct {
	PlayerID        string
	OldPartitionID  string
	NewPartitionID  string
	OldPartitionIDs []string
	ChangeType      PartitionChangeType
}

// PartitionChangeType 分区变化类型
type PartitionChangeType int

const (
	PartitionSplit PartitionChangeType = iota
	PartitionMerge
	PartitionResize
)
