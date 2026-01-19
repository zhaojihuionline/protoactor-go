package core

/*

 */

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
	"github.com/asynkron/protoactor-go/slg-game/utils"
	"github.com/asynkron/protoactor-go/slg-game/utils/calc"
)

type PlayerActor struct {
	actor.Actor
	Name                 string
	CurPosition          bmap.Position
	CurScale             float64
	CurLayerNumber       bmap.LayerNumber
	CurAOIPartions       map[bmap.LayerNumber]map[utils.BigmapCoord]bool
	LastAOICache         *bmap.AOICache
	LastLayerEntityCache map[bmap.LayerNumber]*map[bmap.EntityID]*bmap.Entity
	PartitionPIDs        map[utils.BigmapCoord]*actor.PID
}

func NewPlayerActor(name string, partitionPIDs map[utils.BigmapCoord]*actor.PID) *PlayerActor {
	return &PlayerActor{
		Name:           name,
		CurAOIPartions: make(map[bmap.LayerNumber]map[utils.BigmapCoord]bool),
		PartitionPIDs:  partitionPIDs,
	}
}

// partitionBounds 根据分区ID计算分区边界 (minX, minY, maxX, maxY)
func partitionBounds(partitionID utils.BigmapCoord) (int32, int32, int32, int32) {
	return GetPartitionBounds(partitionID)
}

// rectsIntersect 检查两个矩形是否相交
func rectsIntersect(aMinX, aMinY, aMaxX, aMaxY, bMinX, bMinY, bMaxX, bMaxY int32) bool {
	return !(aMaxX < bMinX || aMinX > bMaxX || aMaxY < bMinY || aMinY > bMaxY)
}

// ensureCurAOIMap 确保当前层的AOI分区映射存在
func (a *PlayerActor) ensureCurAOIMap(layer bmap.LayerNumber) {
	if a.CurAOIPartions == nil {
		a.CurAOIPartions = make(map[bmap.LayerNumber]map[utils.BigmapCoord]bool)
	}
	if _, ok := a.CurAOIPartions[layer]; !ok {
		a.CurAOIPartions[layer] = make(map[utils.BigmapCoord]bool)
	}
}

// handleAOIUpdate 处理AOI更新，计算新的订阅集合
func (a *PlayerActor) handleAOIUpdate(context actor.Context, layer bmap.LayerNumber, center bmap.Position, view bmap.View) {
	// 计算视野覆盖的候选分区
	possible := calc.ComputePartitionsForAOI(center, view)

	// 计算AOI矩形边界
	aoiMinX := center.X - view.W
	aoiMaxX := center.X + view.W
	aoiMinY := center.Y - view.H
	aoiMaxY := center.Y + view.H

	// 确保当前层映射存在
	a.ensureCurAOIMap(layer)
	current := a.CurAOIPartions[layer]

	// 计算新的订阅集合 (基于分区ID)
	newSubscribed := make(map[utils.BigmapCoord]bool)
	for _, partitionID := range possible {
		// 检查AOI与分区空间相交
		partMinX, partMinY, partMaxX, partMaxY := partitionBounds(partitionID)
		if rectsIntersect(aoiMinX, aoiMinY, aoiMaxX, aoiMaxY, partMinX, partMinY, partMaxX, partMaxY) {
			newSubscribed[partitionID] = true
			// 如果当前没有订阅，发送订阅消息
			if !current[partitionID] {
				pid := a.PartitionPIDs[partitionID]
				if pid != nil {
					context.Send(pid, &bmap.SubscribePlayer{Layer: layer, PID: context.Self()})
					current[partitionID] = true
				}
			}
		}
	}

	// 取消不再需要的订阅
	for partitionID := range current {
		if !newSubscribed[partitionID] {
			pid := a.PartitionPIDs[partitionID]
			if pid != nil {
				context.Send(pid, &bmap.UnsubscribePlayer{Layer: layer, PID: context.Self()})
			}
			delete(current, partitionID)
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
		fmt.Printf("PlayerActor entering map at layer %d, position (%d, %d)\n", msg.Layer, msg.Center.X, msg.Center.Y)
		a.CurLayerNumber = msg.Layer
		a.CurPosition = msg.Center
		a.ensureCurAOIMap(msg.Layer)
		a.handleAOIUpdate(context, msg.Layer, msg.Center, msg.View)

	case *bmap.MoveView:
		fmt.Printf("PlayerActor moving view at layer %d, position (%d, %d)\n", msg.Layer, msg.Center.X, msg.Center.Y)
		a.CurPosition = msg.Center
		a.handleAOIUpdate(context, msg.Layer, msg.Center, msg.View)

	case *bmap.LeaveMap:
		fmt.Printf("PlayerActor leaving map at layer %s\n", context.Self().Id)
		for layer := bmap.LayerNumber(1); layer <= bmap.LayerCount; layer++ {
			if layerMap, ok := a.CurAOIPartions[layer]; ok {
				for partitionID := range layerMap {
					pid := a.PartitionPIDs[partitionID]
					if pid != nil {
						context.Send(pid, &bmap.UnsubscribePlayer{Layer: layer, PID: context.Self()})
					}
				}
				delete(a.CurAOIPartions, layer)
			}
		}
	default:
		_ = msg
	}
}
