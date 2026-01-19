package calc

import (
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
	"github.com/asynkron/protoactor-go/slg-game/utils"
)

// ComputePartitionsForAOI 计算视野范围内可能涉及的分区ID列表
func ComputePartitionsForAOI(center bmap.Position, view bmap.View) []utils.BigmapCoord {
	// view.H 屏幕中心格子坐标的上下方的格子数
	// view.W 屏幕中心格子坐标的左右方的格子数

	// AOI格子
	minX := center.X - view.W
	maxX := center.X + view.W
	minY := center.Y - view.H
	maxY := center.Y + view.H

	// 分区索引范围计算（整数运算）
	pxMin := max(0, minX/bmap.PARTITION_WIDTH)
	pxMax := min(bmap.MAP_WIDTH/bmap.PARTITION_WIDTH-1, maxX/bmap.PARTITION_WIDTH)
	pyMin := max(0, minY/bmap.PARTITION_HEIGHT)
	pyMax := min(bmap.MAP_HEIGHT/bmap.PARTITION_HEIGHT-1, maxY/bmap.PARTITION_HEIGHT)

	// 生成分区列表
	var partitions []utils.BigmapCoord
	for py := pyMin; py <= pyMax; py++ {
		for px := pxMin; px <= pxMax; px++ {
			partitions = append(partitions, utils.EncodeCoord(px, py))
		}
	}
	return partitions
}
