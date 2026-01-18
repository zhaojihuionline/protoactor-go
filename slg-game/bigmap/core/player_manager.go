package core

import (
	"fmt"

	"github.com/asynkron/protoactor-go/actor"
)

// PlayerManager 玩家Actor管理器
type PlayerManager struct {
	system        *actor.ActorSystem
	playerPIDs    map[string]*actor.PID // 玩家ID -> 玩家PID映射
	partitionPIDs map[int64]*actor.PID  // 分区ID -> 分区PID映射
}

// NewPlayerManager 创建新的玩家管理器
func NewPlayerManager(system *actor.ActorSystem, partitionPIDs map[int64]*actor.PID) *PlayerManager {
	return &PlayerManager{
		system:        system,
		playerPIDs:    make(map[string]*actor.PID),
		partitionPIDs: partitionPIDs,
	}
}

// GetOrCreatePlayer 获取或创建玩家Actor
func (pm *PlayerManager) GetOrCreatePlayer(playerID string) *actor.PID {
	// 先检查是否已存在
	if pid, exists := pm.playerPIDs[playerID]; exists {
		return pid
	}

	// 创建新的PlayerActor
	playerProps := actor.PropsFromProducer(func() actor.Actor {
		return NewPlayerActor(playerID, pm.partitionPIDs)
	})

	playerPID, err := pm.system.Root.SpawnNamed(playerProps, fmt.Sprintf("player-%s", playerID))
	if err != nil {
		fmt.Printf("Failed to spawn player actor %s: %v\n", playerID, err)
		return nil
	}

	// 记录到映射中
	pm.playerPIDs[playerID] = playerPID
	fmt.Printf("Created player %s with PID: %s\n", playerID, playerPID.String())

	return playerPID
}

// GetPlayer 获取现有玩家Actor
func (pm *PlayerManager) GetPlayer(playerID string) *actor.PID {
	return pm.playerPIDs[playerID]
}

// RemovePlayer 移除玩家（当玩家断线或离开游戏时调用）
func (pm *PlayerManager) RemovePlayer(playerID string) {
	if pid, exists := pm.playerPIDs[playerID]; exists {
		// 停止PlayerActor
		pm.system.Root.Stop(pid)
		// 从映射中移除
		delete(pm.playerPIDs, playerID)
		fmt.Printf("Removed player %s\n", playerID)
	}
}

// GetAllPlayers 获取所有在线玩家ID列表
func (pm *PlayerManager) GetAllPlayers() []string {
	players := make([]string, 0, len(pm.playerPIDs))
	for playerID := range pm.playerPIDs {
		players = append(players, playerID)
	}
	return players
}

// GetPlayerCount 获取在线玩家数量
func (pm *PlayerManager) GetPlayerCount() int {
	return len(pm.playerPIDs)
}
