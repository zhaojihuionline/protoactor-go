package core

import (
	"fmt"
	"math"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
	"github.com/asynkron/protoactor-go/slg-game/utils/coord"
)

// ValidatePartitionSizes 验证分区尺寸是否能整除地图尺寸
func ValidatePartitionSizes() error {
	if math.Mod(bmap.MAP_WIDTH, bmap.PARTITION_WIDTH) != 0 {
		return fmt.Errorf("partition width %d must evenly divide map width %d", bmap.PARTITION_WIDTH, bmap.MAP_WIDTH)
	}
	if math.Mod(bmap.MAP_HEIGHT, bmap.PARTITION_HEIGHT) != 0 {
		return fmt.Errorf("partition height %d must evenly divide map height %d", bmap.PARTITION_HEIGHT, bmap.MAP_HEIGHT)
	}
	return nil
}

// GetPartitionCounts 获取分区行列数量
func GetPartitionCounts() (perRow, perCol int) {
	perRow = int(bmap.MAP_WIDTH / bmap.PARTITION_WIDTH)
	perCol = int(bmap.MAP_HEIGHT / bmap.PARTITION_HEIGHT)
	return
}

// GetPartitionBounds 根据分区ID获取分区边界坐标
func GetPartitionBounds(partitionID coord.BigmapCoord) (minX, minY, maxX, maxY int32) {
	px, py := partitionID.X(), partitionID.Y()
	minX = px * bmap.PARTITION_WIDTH
	minY = py * bmap.PARTITION_HEIGHT
	maxX = minX + bmap.PARTITION_WIDTH
	maxY = minY + bmap.PARTITION_HEIGHT
	return
}

// PartitionManager 分区管理器，管理分区ID到PID的映射
type PartitionManager struct {
	partitionPIDs map[coord.BigmapCoord]*actor.PID
	pidToID       map[*actor.PID]coord.BigmapCoord
}

// NewPartitionManager 创建新的分区管理器
func NewPartitionManager() *PartitionManager {
	return &PartitionManager{
		partitionPIDs: make(map[coord.BigmapCoord]*actor.PID),
		pidToID:       make(map[*actor.PID]coord.BigmapCoord),
	}
}

// RegisterPartition 注册分区
func (pm *PartitionManager) RegisterPartition(partitionID coord.BigmapCoord, pid *actor.PID) {
	pm.partitionPIDs[partitionID] = pid
	pm.pidToID[pid] = partitionID
}

// GetPartitionPID 根据分区ID获取分区PID
func (pm *PartitionManager) GetPartitionPID(partitionID coord.BigmapCoord) *actor.PID {
	return pm.partitionPIDs[partitionID]
}

// GetPartitionIDByPID 根据PID获取分区ID
func (pm *PartitionManager) GetPartitionIDByPID(pid *actor.PID) coord.BigmapCoord {
	if id, exists := pm.pidToID[pid]; exists {
		return id
	}
	return -1 // 无效ID
}

func (pm *PartitionManager) GetPartionPIDs() map[coord.BigmapCoord]*actor.PID {
	return pm.partitionPIDs
}
