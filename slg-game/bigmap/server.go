package slg

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/bigmap/core"
	"github.com/asynkron/protoactor-go/slg-game/bigmap/logic"
)

// Server 大地图服务器
type Server struct {
	system           *actor.ActorSystem
	partitionManager *core.PartitionManager
	gameWorld        *logic.GameWorld
}

// NewServer 创建新的服务器实例
func NewServer(system *actor.ActorSystem) *Server {
	// 初始化分区
	partitionManager := InitializePartitions(system)

	// 创建游戏世界
	gameWorld := logic.NewGameWorld(system, partitionManager)

	return &Server{
		system:           system,
		partitionManager: partitionManager,
		gameWorld:        gameWorld,
	}
}

// InitializePartitions 初始化所有分区actor并注册到管理器
func InitializePartitions(system *actor.ActorSystem) *core.PartitionManager {
	pm := core.NewPartitionManager()
	const totalPartitions = 144 // 12x12 = 144

	for i := 0; i < totalPartitions; i++ {
		partitionID := int64(i)
		props := actor.PropsFromProducer(func() actor.Actor {
			return &core.PartionActor{ID: partitionID}
		})

		pid, err := system.Root.SpawnNamed(props, fmt.Sprintf("partition-%d", partitionID))
		if err != nil {
			panic(fmt.Sprintf("Failed to spawn partition actor %d: %v", partitionID, err))
		}

		pm.RegisterPartition(partitionID, pid)
		fmt.Printf("Created partition %d\n", partitionID)
	}

	return pm
}

// GetGameWorld 获取游戏世界实例
func (s *Server) GetGameWorld() *logic.GameWorld {
	return s.gameWorld
}
