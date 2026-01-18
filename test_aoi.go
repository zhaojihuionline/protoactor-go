package main

import (
	"fmt"
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
	"github.com/asynkron/protoactor-go/slg-game/bigmap/core"
)

func main() {
	// 测试 AOI 计算
	center := bmap.Position{X: 250, Y: 250}
	view := bmap.View{W: 300, H: 300}

	partitions := core.ComputePartitionsForAOI(center, view)
	fmt.Printf("Center: (%d, %d), View: (%d, %d)\n", center.X, center.Y, view.W, view.H)
	fmt.Printf("Found %d partitions in AOI\n", len(partitions))

	for i, p := range partitions {
		x, y := p.Decode()
		fmt.Printf("Partition %d: (%d, %d) -> ID: %d\n", i, x, y, int64(p))
	}
}