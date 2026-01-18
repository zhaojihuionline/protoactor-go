package core

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

// PartitionManager 分区管理器，管理分区ID到PID的映射
type PartitionManager struct {
	partitionPIDs map[int64]*actor.PID
	pidToID       map[*actor.PID]int64
}

// NewPartitionManager 创建新的分区管理器
func NewPartitionManager() *PartitionManager {
	return &PartitionManager{
		partitionPIDs: make(map[int64]*actor.PID),
		pidToID:       make(map[*actor.PID]int64),
	}
}

// RegisterPartition 注册分区
func (pm *PartitionManager) RegisterPartition(partitionID int64, pid *actor.PID) {
	pm.partitionPIDs[partitionID] = pid
	pm.pidToID[pid] = partitionID
}

// GetPartitionPID 根据分区ID获取分区PID
func (pm *PartitionManager) GetPartitionPID(partitionID int64) *actor.PID {
	return pm.partitionPIDs[partitionID]
}

// GetPartitionIDByPID 根据PID获取分区ID
func (pm *PartitionManager) GetPartitionIDByPID(pid *actor.PID) int64 {
	if id, exists := pm.pidToID[pid]; exists {
		return id
	}
	return -1 // 无效ID
}

// InitializePartitions 初始化所有分区actor并注册到管理器
// 这个函数应该在系统启动时调用
func InitializePartitions(system *actor.ActorSystem) (*PartitionManager, []*actor.PID) {
	pm := NewPartitionManager()
	const totalPartitions = 144 // 12x12 = 144

	partitionPIDs := make([]*actor.PID, totalPartitions)

	for i := 0; i < totalPartitions; i++ {
		partitionID := int64(i)
		props := actor.PropsFromProducer(func() actor.Actor {
			return &PartionActor{ID: partitionID}
		})

		pid, err := system.Root.SpawnNamed(props, fmt.Sprintf("partition-%d", partitionID))
		if err != nil {
			panic(fmt.Sprintf("Failed to spawn partition actor %d: %v", partitionID, err))
		}

		partitionPIDs[i] = pid
		pm.RegisterPartition(partitionID, pid)
	}

	return pm, partitionPIDs
}
