package core

/*

 */

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

type PartionActor struct {
	actor.Actor
	ID        int64
	Listeners map[bmap.LayerNumber]map[*actor.PID]bool
}

func (a *PartionActor) Receive(context actor.Context) {
	switch context.Message().(type) {
	case *actor.Started:
		fmt.Println("PartionActor started")
	case *actor.Stopping:
		fmt.Println("PartionActor stopping")
	case *actor.Stopped:
		fmt.Println("PartionActor stopped")
	case *actor.Restarting:
		fmt.Println("PartionActor restarting")
	case *actor.Stop:
		fmt.Println("PartionActor received stop message")
	case *actor.Failure:
		fmt.Println("PartionActor received failure message")
	case *actor.Terminated:
		fmt.Println("PartionActor received terminated message")
	case *actor.Watch:
		fmt.Println("PartionActor received watch message")
	case *actor.Unwatch:
		fmt.Println("PartionActor received unwatch message")
	}
}
