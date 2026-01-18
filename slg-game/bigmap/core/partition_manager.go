package core

import (
	"fmt"
	"math"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/utils"
)

// 地图和分区尺寸常量
const (
	MAP_WIDTH        = 1200 // 地图宽度
	MAP_HEIGHT       = 1200 // 地图高度
	PARTITION_WIDTH  = 100  // 分区宽度 (必须能整除MAP_WIDTH)
	PARTITION_HEIGHT = 100  // 分区高度 (必须能整除MAP_HEIGHT)
)

// ValidatePartitionSizes 验证分区尺寸是否能整除地图尺寸
func ValidatePartitionSizes() error {
	if math.Mod(MAP_WIDTH, PARTITION_WIDTH) != 0 {
		return fmt.Errorf("partition width %d must evenly divide map width %d", PARTITION_WIDTH, MAP_WIDTH)
	}
	if math.Mod(MAP_HEIGHT, PARTITION_HEIGHT) != 0 {
		return fmt.Errorf("partition height %d must evenly divide map height %d", PARTITION_HEIGHT, MAP_HEIGHT)
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
func GetPartitionBounds(partitionID utils.BigmapCoord) (minX, minY, maxX, maxY float64) {
	px, py := partitionID.X(), partitionID.Y()
	minX = float64(px) * PARTITION_WIDTH
	minY = float64(py) * PARTITION_HEIGHT
	maxX = minX + PARTITION_WIDTH
	maxY = minY + PARTITION_HEIGHT
	return
}

// PartitionManager 分区管理器，管理分区ID到PID的映射
type PartitionManager struct {
	partitionPIDs map[utils.BigmapCoord]*actor.PID
	pidToID       map[*actor.PID]utils.BigmapCoord
}

// NewPartitionManager 创建新的分区管理器
func NewPartitionManager() *PartitionManager {
	return &PartitionManager{
		partitionPIDs: make(map[utils.BigmapCoord]*actor.PID),
		pidToID:       make(map[*actor.PID]utils.BigmapCoord),
	}
}

// RegisterPartition 注册分区
func (pm *PartitionManager) RegisterPartition(partitionID utils.BigmapCoord, pid *actor.PID) {
	pm.partitionPIDs[partitionID] = pid
	pm.pidToID[pid] = partitionID
}

// GetPartitionPID 根据分区ID获取分区PID
func (pm *PartitionManager) GetPartitionPID(partitionID utils.BigmapCoord) *actor.PID {
	return pm.partitionPIDs[partitionID]
}

// GetPartitionIDByPID 根据PID获取分区ID
func (pm *PartitionManager) GetPartitionIDByPID(pid *actor.PID) utils.BigmapCoord {
	if id, exists := pm.pidToID[pid]; exists {
		return id
	}
	return -1 // 无效ID
}

func (pm *PartitionManager) GetPartionPIDs() map[utils.BigmapCoord]*actor.PID {
	return pm.partitionPIDs
}
