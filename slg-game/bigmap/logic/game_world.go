package logic

import (
	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/bigmap/core"
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

// GameWorld 大地图游戏世界管理器
type GameWorld struct {
	system        *actor.ActorSystem
	playerManager *core.PlayerManager
}

// NewGameWorld 创建新的游戏世界
func NewGameWorld(system *actor.ActorSystem, partitionPIDs map[int64]*actor.PID) *GameWorld {
	playerManager := core.NewPlayerManager(system, partitionPIDs)
	return &GameWorld{
		system:        system,
		playerManager: playerManager,
	}
}

// EnterMap 玩家进入大地图接口
func (gw *GameWorld) EnterMap(playerID string, layer bmap.LayerNumber, center bmap.Position, view bmap.View) error {
	playerPID := gw.playerManager.GetOrCreatePlayer(playerID)
	gw.system.Root.Send(playerPID, &bmap.EnterMap{
		Layer:  layer,
		Center: center,
		View:   view,
	})
	return nil
}

// MoveView 玩家视野移动接口
func (gw *GameWorld) MoveView(playerID string, layer bmap.LayerNumber, center bmap.Position, view bmap.View) error {
	playerPID := gw.playerManager.GetPlayer(playerID)
	if playerPID == nil {
		return gw.EnterMap(playerID, layer, center, view) // 如果玩家不存在，先进入地图
	}
	gw.system.Root.Send(playerPID, &bmap.MoveView{
		Layer:  layer,
		Center: center,
		View:   view,
	})
	return nil
}

// LeaveMap 玩家离开大地图接口
func (gw *GameWorld) LeaveMap(playerID string, layer bmap.LayerNumber) error {
	playerPID := gw.playerManager.GetPlayer(playerID)
	if playerPID == nil {
		return nil // 玩家不存在，无需离开
	}
	gw.system.Root.Send(playerPID, &bmap.LeaveMap{
		Layer: layer,
	})
	return nil
}
