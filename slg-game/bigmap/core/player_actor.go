package core

/*

 */

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

type PlayerActor struct {
	actor.Actor
	Name                 string
	CurPosition          bmap.Position
	CurScale             float64
	CurLayerNumber       bmap.LayerNumber
	LastAOICache         *bmap.AOICache
	LastLayerEntityCache map[bmap.LayerNumber]*map[bmap.EntityID]*bmap.Entity
}

func (a *PlayerActor) Receive(context actor.Context) {
	switch context.Message().(type) {
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
	}
}
