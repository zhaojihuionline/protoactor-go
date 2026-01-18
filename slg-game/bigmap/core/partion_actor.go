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
	Position      bmap.Position
	CurAOIPlayers map[bmap.LayerNumber]map[*actor.PID]bool
}

func (a *PartionActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		fmt.Printf("PartionActor %s started\n", context.Self().Id)
		if a.CurAOIPlayers == nil {
			a.CurAOIPlayers = make(map[bmap.LayerNumber]map[*actor.PID]bool)
		}
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

	// 订阅/退订处理（来自 PlayerActor）
	case *bmap.SubscribePlayer:
		if a.CurAOIPlayers[msg.Layer] == nil {
			a.CurAOIPlayers[msg.Layer] = make(map[*actor.PID]bool)
		}
		a.CurAOIPlayers[msg.Layer][msg.PID] = true
		fmt.Printf("PartionActor %s: subscribed player %s on layer %d\n", context.Self().Id, msg.PID.String(), msg.Layer)

	case *bmap.UnsubscribePlayer:
		if layerPlayers, ok := a.CurAOIPlayers[msg.Layer]; ok {
			delete(layerPlayers, msg.PID)
			if len(layerPlayers) == 0 {
				delete(a.CurAOIPlayers, msg.Layer)
			}
		}
		fmt.Printf("PartionActor %s: unsubscribed player %s on layer %d\n", context.Self().Id, msg.PID.String(), msg.Layer)
	}
}
