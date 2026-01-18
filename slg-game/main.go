package main

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/bigmap/core"
	"github.com/asynkron/protoactor-go/slg-game/domain/bmap"
)

func main() {
	// 创建actor系统
	system := actor.NewActorSystem()

	// 初始化144个分区actor (12x12 = 144)
	partitionPIDs := make(map[int64]*actor.PID)
	for i := 0; i < 144; i++ {
		partitionID := int64(i)
		props := actor.PropsFromProducer(func() actor.Actor {
			return &core.PartionActor{ID: partitionID}
		})

		pid, err := system.Root.SpawnNamed(props, fmt.Sprintf("partition-%d", partitionID))
		if err != nil {
			panic(fmt.Sprintf("Failed to spawn partition actor %d: %v", partitionID, err))
		}

		partitionPIDs[partitionID] = pid
		fmt.Printf("Created partition %d with PID: %s\n", partitionID, pid.String())
	}

	// 创建玩家actor，注入分区PID映射
	playerProps := actor.PropsFromProducer(func() actor.Actor {
		return core.NewPlayerActor("player1", partitionPIDs)
	})

	playerPID, err := system.Root.SpawnNamed(playerProps, "player1")
	if err != nil {
		panic(fmt.Sprintf("Failed to spawn player actor: %v", err))
	}

	fmt.Printf("Created player with PID: %s\n", playerPID.String())

	// 示例：发送EnterMap消息
	system.Root.Send(playerPID, &bmap.EnterMap{
		Layer:  1,
		Center: bmap.Position{X: 250, Y: 250}, // 视野中心在分区(2,2)和周围分区
		View:   bmap.View{W: 300, H: 300},     // 视野大小300x300
	})

	// 等待系统运行
	fmt.Println("System is running. Press Ctrl+C to exit.")
	select {}
}
