package slg

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/asynkron/protoactor-go/slg-game/bigmap/core"
	"github.com/asynkron/protoactor-go/slg-game/bigmap/logic"
	"github.com/asynkron/protoactor-go/slg-game/utils"
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
	// 验证分区尺寸
	if err := core.ValidatePartitionSizes(); err != nil {
		panic(fmt.Sprintf("Invalid partition sizes: %v", err))
	}

	pm := core.NewPartitionManager()
	partitionsPerRow, partitionsPerCol := core.GetPartitionCounts()

	fmt.Printf("Initializing %dx%d partitions (total: %d)\n", partitionsPerRow, partitionsPerCol, partitionsPerRow*partitionsPerCol)
	fmt.Printf("Map size: %dx%d, Partition size: %dx%d\n",
		core.MAP_WIDTH, core.MAP_HEIGHT, core.PARTITION_WIDTH, core.PARTITION_HEIGHT)

	// 创建分区，按照地图坐标顺序 (0,0), (100,0), (200,0), ... 到 (1100,1100)
	for mapY := 0; mapY < core.MAP_HEIGHT; mapY += core.PARTITION_HEIGHT {
		for mapX := 0; mapX < core.MAP_WIDTH; mapX += core.PARTITION_WIDTH {
			// 计算分区索引用于编码
			partitionIndexX := mapX / core.PARTITION_WIDTH
			partitionIndexY := mapY / core.PARTITION_HEIGHT

			partitionID := int64(utils.EncodeCoord(partitionIndexX, partitionIndexY))
			props := actor.PropsFromProducer(func() actor.Actor {
				return &core.PartionActor{ID: partitionID}
			})

			pid, err := system.Root.SpawnNamed(props, fmt.Sprintf("partition-%d-%d", mapX, mapY))
			if err != nil {
				panic(fmt.Sprintf("Failed to spawn partition actor (%d,%d): %v", mapX, mapY, err))
			}

			pm.RegisterPartition(partitionID, pid)
			fmt.Printf("Created partition at map(%d,%d) with index(%d,%d) ID=%d\n",
				mapX, mapY, partitionIndexX, partitionIndexY, partitionID)
		}
	}

	return pm
}

// GetGameWorld 获取游戏世界实例
func (s *Server) GetGameWorld() *logic.GameWorld {
	return s.gameWorld
}
