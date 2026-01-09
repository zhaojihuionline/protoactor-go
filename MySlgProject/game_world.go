package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// GameWorld 游戏世界管理器
type GameWorld struct {
	system *actor.ActorSystem
	config *Config

	// 核心组件
	spatialIndex *SpatialIndex
	partitionMgr *PartitionManager
	entityMgr    *EntityManager
	eventSystem  *EventSystem
	stateSyncMgr *StateSyncManager

	// 玩家和AOI管理
	playerAOIManagers map[string]*AOIManager
	playerSubscribers map[string]*AOISubscriber

	// 世界状态
	isRunning   bool
	startTime   time.Time
	lastUpdate  time.Time
	updateMutex sync.RWMutex

	// 统计信息
	stats WorldStats
}

// WorldStats 世界统计信息
type WorldStats struct {
	TotalPlayers    int
	TotalEntities   int
	TotalPartitions int
	Uptime          time.Duration
	LastUpdate      time.Time
}

// PartitionManager 分区管理器
type PartitionManager struct {
	partitions   map[string]*Partition
	mutex        sync.RWMutex
	spatialIndex *SpatialIndex
}

// NewGameWorld 创建游戏世界
func NewGameWorld(config *Config) *GameWorld {
	system := actor.NewActorSystem()

	// 初始化核心组件
	spatialIndex := NewSpatialIndex(config)
	partitionMgr := &PartitionManager{
		partitions:   make(map[string]*Partition),
		spatialIndex: spatialIndex,
	}
	entityMgr := NewEntityManager(spatialIndex, nil, system) // eventSystem稍后设置
	eventSystem := NewEventSystem(system, 1000)
	stateSyncMgr := NewStateSyncManager(partitionMgr, system, 100, time.Second*1)

	// 设置事件系统引用
	entityMgr.eventSystem = eventSystem

	return &GameWorld{
		system:            system,
		config:            config,
		spatialIndex:      spatialIndex,
		partitionMgr:      partitionMgr,
		entityMgr:         entityMgr,
		eventSystem:       eventSystem,
		stateSyncMgr:      stateSyncMgr,
		playerAOIManagers: make(map[string]*AOIManager),
		playerSubscribers: make(map[string]*AOISubscriber),
		isRunning:         false,
	}
}

// Start 启动游戏世界
func (gw *GameWorld) Start() error {
	gw.updateMutex.Lock()
	defer gw.updateMutex.Unlock()

	if gw.isRunning {
		return fmt.Errorf("game world is already running")
	}

	log.Println("Starting game world...")

	// 初始化地图分区
	gw.initializePartitions()

	// 初始化一些测试实体
	gw.initializeTestEntities()

	// 启动世界更新循环
	go gw.worldUpdateLoop()

	gw.isRunning = true
	gw.startTime = time.Now()
	gw.lastUpdate = time.Now()

	log.Println("Game world started successfully")
	return nil
}

// Stop 停止游戏世界
func (gw *GameWorld) Stop() error {
	gw.updateMutex.Lock()
	defer gw.updateMutex.Unlock()

	if !gw.isRunning {
		return fmt.Errorf("game world is not running")
	}

	log.Println("Stopping game world...")

	gw.isRunning = false
	log.Println("Game world stopped")
	return nil
}

// initializePartitions 初始化地图分区
func (gw *GameWorld) initializePartitions() {
	// 使用固定的100x100分区，创建12x12=144个分区

	for i := 0; i < PartitionsPerRow; i++ {
		for j := 0; j < PartitionsPerRow; j++ {
			partitionID := fmt.Sprintf("region_%d_%d", j, i)

			bounds := Rectangle{
				X:      j * PartitionSize,
				Y:      i * PartitionSize,
				Width:  PartitionSize,
				Height: PartitionSize,
			}

			partition := NewPartition(partitionID, bounds, gw.config, gw.system)
			partition.Initialize()

			gw.partitionMgr.AddPartition(partition)
		}
	}

	log.Printf("Initialized %d fixed partitions (%dx%d grid)", len(gw.partitionMgr.partitions), PartitionsPerRow, PartitionsPerRow)
}

// initializeTestEntities 初始化测试实体
func (gw *GameWorld) initializeTestEntities() {
	// 创建一些测试NPC
	npcs := []struct {
		id      string
		name    string
		pos     Point
		npcType NPCType
	}{
		{"npc_001", "村民小明", Point{X: 100, Y: 100}, NPCVillager},
		{"npc_002", "守卫队长", Point{X: 200, Y: 200}, NPCGuard},
		{"npc_003", "商人老王", Point{X: 300, Y: 300}, NPCTrader},
		{"npc_004", "任务发布员", Point{X: 400, Y: 400}, NPCQuestGiver},
	}

	for _, npcData := range npcs {
		npc := NewNPC(npcData.id, npcData.name, npcData.npcType, npcData.pos)
		if err := gw.entityMgr.AddEntity(npc); err != nil {
			log.Printf("Failed to add NPC %s: %v", npcData.id, err)
		}
	}

	log.Printf("Initialized %d test NPCs", len(npcs))
}

// worldUpdateLoop 世界更新循环
func (gw *GameWorld) worldUpdateLoop() {
	ticker := time.NewTicker(time.Second / 30) // 30 FPS
	defer ticker.Stop()

	lastUpdate := time.Now()

	for range ticker.C {
		if !gw.isRunning {
			break
		}

		now := time.Now()
		deltaTime := now.Sub(lastUpdate)
		lastUpdate = now

		gw.update(deltaTime)
	}
}

// update 更新世界状态
func (gw *GameWorld) update(deltaTime time.Duration) {
	gw.updateMutex.Lock()
	defer gw.updateMutex.Unlock()

	// 更新所有NPC
	gw.updateNPCs(deltaTime)

	// 更新AOI管理器
	gw.updateAOIManagers()

	// 清理过期订阅
	gw.eventSystem.Cleanup()

	// 更新统计信息
	gw.updateStats()

	gw.lastUpdate = time.Now()
}

// updateNPCs 更新NPC状态
func (gw *GameWorld) updateNPCs(deltaTime time.Duration) {
	npcs := gw.entityMgr.GetEntitiesByType(EntityNPC)
	for _, entity := range npcs {
		if npc, ok := entity.(*NPC); ok {
			npc.Update(deltaTime)
		}
	}
}

// updateAOIManagers 更新AOI管理器
func (gw *GameWorld) updateAOIManagers() {
	for _, aoiMgr := range gw.playerAOIManagers {
		aoiMgr.CleanupExpiredCache()
	}
}

// updateStats 更新统计信息
func (gw *GameWorld) updateStats() {
	gw.stats.TotalPlayers = len(gw.playerAOIManagers)
	gw.stats.TotalEntities = gw.entityMgr.GetEntityCount()
	gw.stats.TotalPartitions = len(gw.partitionMgr.partitions)

	if gw.isRunning {
		gw.stats.Uptime = time.Since(gw.startTime)
	}

	gw.stats.LastUpdate = time.Now()
}

// AddPlayer 添加玩家
func (gw *GameWorld) AddPlayer(playerID, playerName string, startPos Point) (*Player, error) {
	// 创建玩家实体
	player := NewPlayer(playerID, playerName, startPos)

	// 添加到实体管理器
	if err := gw.entityMgr.AddEntity(player); err != nil {
		return nil, err
	}

	// 创建AOI管理器
	aoiManager := NewAOIManager(playerID, startPos, gw.config, gw.system,
		gw.spatialIndex, gw.partitionMgr)

	// 创建AOI订阅器
	aoiSubscriber := NewAOISubscriber(playerID, nil, gw.config, gw.system, gw.partitionMgr)

	// 存储引用
	gw.playerAOIManagers[playerID] = aoiManager
	gw.playerSubscribers[playerID] = aoiSubscriber

	// 发布玩家加入事件
	gw.eventSystem.Publish(NewPlayerJoinEvent(playerID, playerName, startPos))

	log.Printf("Player %s joined at position (%d,%d)", playerID, startPos.X, startPos.Y)
	return player, nil
}

// RemovePlayer 移除玩家
func (gw *GameWorld) RemovePlayer(playerID string) error {
	// 从实体管理器移除
	if err := gw.entityMgr.RemoveEntity(playerID); err != nil {
		return err
	}

	// 清理AOI管理器和订阅器
	delete(gw.playerAOIManagers, playerID)
	delete(gw.playerSubscribers, playerID)

	// 发布玩家离开事件
	gw.eventSystem.Publish(&PlayerLeaveEvent{
		BaseEvent: BaseEvent{
			EventType: "player_leave",
			Timestamp: time.Now(),
			SourceID:  playerID,
		},
		PlayerID: playerID,
		Reason:   "normal_leave",
	})

	log.Printf("Player %s left the game", playerID)
	return nil
}

// UpdatePlayerPosition 更新玩家位置
func (gw *GameWorld) UpdatePlayerPosition(playerID string, newPos Point) error {
	// 更新实体位置
	if err := gw.entityMgr.UpdateEntityPosition(playerID, newPos); err != nil {
		return err
	}

	// 更新AOI管理器
	if aoiMgr, exists := gw.playerAOIManagers[playerID]; exists {
		aoiMgr.UpdatePosition(newPos)
	}

	return nil
}

// GetPlayer 获取玩家
func (gw *GameWorld) GetPlayer(playerID string) (*Player, bool) {
	entity, exists := gw.entityMgr.GetEntity(playerID)
	if !exists {
		return nil, false
	}

	player, ok := entity.(*Player)
	return player, ok
}

// GetWorldStats 获取世界统计信息
func (gw *GameWorld) GetWorldStats() WorldStats {
	gw.updateMutex.RLock()
	defer gw.updateMutex.RUnlock()

	return gw.stats
}

// GetSystemStats 获取系统统计信息
func (gw *GameWorld) GetSystemStats() map[string]interface{} {
	stats := map[string]interface{}{
		"world":    gw.GetWorldStats(),
		"entities": gw.entityMgr.GetEntityStats(),
		"events":   gw.eventSystem.GetEventStats(),
		"sync":     gw.stateSyncMgr.GetSyncStats(),
	}

	return stats
}

// Shutdown 关闭世界
func (gw *GameWorld) Shutdown() {
	log.Println("Shutting down game world...")

	if gw.isRunning {
		gw.Stop()
	}

	// 等待一段时间确保所有操作完成
	time.Sleep(time.Second * 2)

	log.Println("Game world shutdown complete")
}

// PartitionManager 方法

// AddPartition 添加分区
func (pm *PartitionManager) AddPartition(partition *Partition) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.partitions[partition.ID] = partition
}

// RemovePartition 移除分区
func (pm *PartitionManager) RemovePartition(partitionID string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	delete(pm.partitions, partitionID)
}

// GetPartition 获取分区
func (pm *PartitionManager) GetPartition(partitionID string) (*Partition, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	partition, exists := pm.partitions[partitionID]
	return partition, exists
}

// GetAllPartitions 获取所有分区
func (pm *PartitionManager) GetAllPartitions() map[string]*Partition {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result := make(map[string]*Partition)
	for id, partition := range pm.partitions {
		result[id] = partition
	}

	return result
}

// GetPartitionCount 获取分区数量
func (pm *PartitionManager) GetPartitionCount() int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return len(pm.partitions)
}
