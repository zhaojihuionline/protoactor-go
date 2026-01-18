package core

import (
	"github.com/asynkron/protoactor-go/actor"
)

// 分区ID编码常量 (int64位移编码)
// 使用12位坐标空间，支持4096x4096分区
const (
	PARTITION_BITS = 12                        // 坐标位数
	PARTITION_MASK = (1 << PARTITION_BITS) - 1 // 坐标掩码 (4095)
	PARTITION_MAX  = PARTITION_MASK            // 最大坐标值
)

// EncodePartitionID 将分区坐标编码为int64 ID
func EncodePartitionID(px, py int) int64 {
	return int64(py)<<PARTITION_BITS | int64(px)
}

// DecodePartitionID 从int64 ID解码出分区坐标
func DecodePartitionID(id int64) (px, py int) {
	return int(id & PARTITION_MASK), int(id >> PARTITION_BITS)
}

// GetPartitionBounds 根据分区ID获取分区边界坐标
func GetPartitionBounds(partitionID int64) (minX, minY, maxX, maxY float64) {
	const partitionSize = 100.0
	px, py := DecodePartitionID(partitionID)
	minX = float64(px) * partitionSize
	minY = float64(py) * partitionSize
	maxX = minX + partitionSize
	maxY = minY + partitionSize
	return
}

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

func (pm *PartitionManager) GetPartionPIDs() map[int64]*actor.PID {
	return pm.partitionPIDs
}
