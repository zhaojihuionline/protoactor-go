package main

import (
	"fmt"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	slg "github.com/asynkron/protoactor-go/slg-game/bigmap"
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

func main() {
	// 创建actor系统
	system := actor.NewActorSystem()

	// 创建大地图服务器（包含分区初始化和游戏世界）
	server := slg.NewServer(system)
	gameWorld := server.GetGameWorld()

	// 演示业务接口使用
	playerID := "player1"

	// 1. 玩家进入大地图
	fmt.Printf("\n=== 玩家 %s 进入大地图 ===\n", playerID)
	err := gameWorld.EnterMap(playerID, 1, bmap.Position{X: 250, Y: 250}, bmap.View{W: 300, H: 300})
	if err != nil {
		fmt.Printf("EnterMap failed: %v\n", err)
	}

	// 等待消息处理
	time.Sleep(100 * time.Millisecond)

	// 2. 玩家移动视野
	fmt.Printf("\n=== 玩家 %s 视野移动 ===\n", playerID)
	err = gameWorld.MoveView(playerID, 1, bmap.Position{X: 450, Y: 450}, bmap.View{W: 300, H: 300})
	if err != nil {
		fmt.Printf("MoveView failed: %v\n", err)
	}

	// 等待消息处理
	time.Sleep(100 * time.Millisecond)

	// 3. 玩家离开大地图
	fmt.Printf("\n=== 玩家 %s 离开大地图 ===\n", playerID)
	err = gameWorld.LeaveMap(playerID, 1)
	if err != nil {
		fmt.Printf("LeaveMap failed: %v\n", err)
	}

	// 显示统计信息
	time.Sleep(100 * time.Millisecond)
	fmt.Printf("\n=== 系统统计 ===\n")
	fmt.Printf("在线玩家数量: %d\n", gameWorld.GetPlayerManager().GetPlayerCount())
	fmt.Printf("在线玩家列表: %v\n", gameWorld.GetPlayerManager().GetAllPlayers())

	// 等待系统运行
	fmt.Println("\nSystem is running. Press Ctrl+C to exit.")
	select {}
}
