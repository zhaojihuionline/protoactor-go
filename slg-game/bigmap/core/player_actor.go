package core

/*

 */

import (
	"fmt"
	"math"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

type PlayerActor struct {
	actor.Actor
	Name                 string
	CurPosition          bmap.Position
	CurScale             float64
	CurLayerNumber       bmap.LayerNumber
	CurAOIPartions       map[bmap.LayerNumber]map[*actor.PID]bool
	LastAOICache         *bmap.AOICache
	LastLayerEntityCache map[bmap.LayerNumber]*map[bmap.EntityID]*bmap.Entity
	PartitionPIDs        map[int64]*actor.PID
}

func NewPlayerActor(name string, partitionPIDs map[int64]*actor.PID) *PlayerActor {
	return &PlayerActor{
		Name:           name,
		CurAOIPartions: make(map[bmap.LayerNumber]map[*actor.PID]bool),
		PartitionPIDs:  partitionPIDs,
	}
}

// computePartitionsForAOI 计算视野范围内可能涉及的分区ID列表
func computePartitionsForAOI(center bmap.Position, view bmap.View) []int64 {
	const partitionSize = 100.0

	halfW := float64(view.W) / 2
	halfH := float64(view.H) / 2

	minX := center.X - halfW
	maxX := center.X + halfW
	minY := center.Y - halfH
	maxY := center.Y + halfH

	// 计算分区范围，限制在地图边界内
	pxMin := int(math.Max(0, math.Floor(minX/partitionSize)))
	pxMax := int(math.Min(PARTITION_MAX, math.Floor(maxX/partitionSize)))
	pyMin := int(math.Max(0, math.Floor(minY/partitionSize)))
	pyMax := int(math.Min(PARTITION_MAX, math.Floor(maxY/partitionSize)))

	var partitions []int64
	for py := pyMin; py <= pyMax; py++ {
		for px := pxMin; px <= pxMax; px++ {
			partitionID := EncodePartitionID(px, py)
			partitions = append(partitions, partitionID)
		}
	}

	return partitions
}

// partitionBounds 根据分区ID计算分区边界 (minX, minY, maxX, maxY)
func partitionBounds(partitionID int64) (float64, float64, float64, float64) {
	return GetPartitionBounds(partitionID)
}

// rectsIntersect 检查两个矩形是否相交
func rectsIntersect(aMinX, aMinY, aMaxX, aMaxY, bMinX, bMinY, bMaxX, bMaxY float64) bool {
	return !(aMaxX < bMinX || aMinX > bMaxX || aMaxY < bMinY || aMinY > bMaxY)
}

// ensureCurAOIMap 确保当前层的AOI分区映射存在
func (a *PlayerActor) ensureCurAOIMap(layer bmap.LayerNumber) {
	if a.CurAOIPartions == nil {
		a.CurAOIPartions = make(map[bmap.LayerNumber]map[*actor.PID]bool)
	}
	if _, ok := a.CurAOIPartions[layer]; !ok {
		a.CurAOIPartions[layer] = make(map[*actor.PID]bool)
	}
}

// handleAOIUpdate 处理AOI更新，计算新的订阅集合
func (a *PlayerActor) handleAOIUpdate(context actor.Context, layer bmap.LayerNumber, center bmap.Position, view bmap.View) {
	// 计算视野覆盖的候选分区
	possible := computePartitionsForAOI(center, view)

	// 计算AOI矩形边界
	halfW := float64(view.W) / 2
	halfH := float64(view.H) / 2
	aoiMinX := center.X - halfW
	aoiMaxX := center.X + halfW
	aoiMinY := center.Y - halfH
	aoiMaxY := center.Y + halfH

	// 确保当前层映射存在
	a.ensureCurAOIMap(layer)
	current := a.CurAOIPartions[layer]

	// 计算新的订阅集合
	newSet := make(map[*actor.PID]bool)
	for _, partitionID := range possible {
		pid := a.PartitionPIDs[partitionID]
		if pid == nil {
			continue
		}

		// 检查AOI与分区空间相交
		partMinX, partMinY, partMaxX, partMaxY := partitionBounds(partitionID)
		if rectsIntersect(aoiMinX, aoiMinY, aoiMaxX, aoiMaxY, partMinX, partMinY, partMaxX, partMaxY) {
			newSet[pid] = true
			// 如果当前没有订阅，发送订阅消息
			if !current[pid] {
				context.Send(pid, &bmap.SubscribePlayer{Layer: layer, PID: context.Self()})
				current[pid] = true
			}
		}
	}

	// 取消不再需要的订阅
	for pid := range current {
		if !newSet[pid] {
			context.Send(pid, &bmap.UnsubscribePlayer{Layer: layer, PID: context.Self()})
			delete(current, pid)
		}
	}
}

func (a *PlayerActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		fmt.Println("PlayerActor started")
	case *actor.Stopping:
		fmt.Println("PlayerActor stopping")
	case *actor.Stopped:
		fmt.Println("PlayerActor stopped")
	case *actor.Restarting:
		fmt.Println("PlayerActor restarting")
	case *actor.Stop:
		fmt.Println("PlayerActor received stop message")
	case *actor.Failure:
		fmt.Println("PlayerActor received failure message")
	case *actor.Terminated:
		fmt.Println("PlayerActor received terminated message")
	case *actor.Watch:
		fmt.Println("PlayerActor received watch message")
	case *actor.Unwatch:
		fmt.Println("PlayerActor received unwatch message")

	case *bmap.EnterMap:
		fmt.Printf("PlayerActor entering map at layer %d, position (%f, %f)\n", msg.Layer, msg.Center.X, msg.Center.Y)
		a.CurLayerNumber = msg.Layer
		a.CurPosition = msg.Center
		a.ensureCurAOIMap(msg.Layer)
		a.handleAOIUpdate(context, msg.Layer, msg.Center, msg.View)

	case *bmap.MoveView:
		fmt.Printf("PlayerActor moving view at layer %d, position (%f, %f)\n", msg.Layer, msg.Center.X, msg.Center.Y)
		a.CurPosition = msg.Center
		a.handleAOIUpdate(context, msg.Layer, msg.Center, msg.View)

	case *bmap.LeaveMap:
		fmt.Printf("PlayerActor leaving map at layer %d\n", msg.Layer)
		if layerMap, ok := a.CurAOIPartions[msg.Layer]; ok {
			for pid := range layerMap {
				context.Send(pid, &bmap.UnsubscribePlayer{Layer: msg.Layer, PID: context.Self()})
			}
			delete(a.CurAOIPartions, msg.Layer)
		}
	default:
		_ = msg
	}
}
