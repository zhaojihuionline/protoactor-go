package core

import (
	"fmt"
	"math"

	"github.com/asynkron/protoactor-go/actor"
)

// 地图和分区尺寸常量
const (
	MAP_WIDTH        = 1200 // 地图宽度
	MAP_HEIGHT       = 1200 // 地图高度
	PARTITION_WIDTH  = 100  // 分区宽度 (必须能整除MAP_WIDTH)
	PARTITION_HEIGHT = 100  // 分区高度 (必须能整除MAP_HEIGHT)
)

// 分区ID编码常量 (int64位移编码)
// 坐标空间基于分区数量计算
const (
	PARTITION_BITS = 12                        // 坐标位数
	PARTITION_MASK = (1 << PARTITION_BITS) - 1 // 坐标掩码 (4095)
)

// EncodePartitionID 将分区坐标编码为int64 ID
func EncodePartitionID(px, py int) int64 {
	return int64(py)<<PARTITION_BITS | int64(px)
}

// DecodePartitionID 从int64 ID解码出分区坐标
func DecodePartitionID(id int64) (px, py int) {
	return int(id & PARTITION_MASK), int(id >> PARTITION_BITS)
}

// ValidatePartitionSizes 验证分区尺寸是否能整除地图尺寸
func ValidatePartitionSizes() error {
	if math.Mod(MAP_WIDTH, PARTITION_WIDTH) != 0 {
		return fmt.Errorf("partition width %.0f must evenly divide map width %.0f", PARTITION_WIDTH, MAP_WIDTH)
	}
	if math.Mod(MAP_HEIGHT, PARTITION_HEIGHT) != 0 {
		return fmt.Errorf("partition height %.0f must evenly divide map height %.0f", PARTITION_HEIGHT, MAP_HEIGHT)
	}
	return nil
}

// GetPartitionCounts 获取分区行列数量
func GetPartitionCounts() (perRow, perCol int) {
	perRow = int(MAP_WIDTH / PARTITION_WIDTH)
	perCol = int(MAP_HEIGHT / PARTITION_HEIGHT)
	return
}

// GetPartitionBounds 根据分区ID获取分区边界坐标
func GetPartitionBounds(partitionID int64) (minX, minY, maxX, maxY float64) {
	px, py := DecodePartitionID(partitionID)
	minX = float64(px) * PARTITION_WIDTH
	minY = float64(py) * PARTITION_HEIGHT
	maxX = minX + PARTITION_WIDTH
	maxY = minY + PARTITION_HEIGHT
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
