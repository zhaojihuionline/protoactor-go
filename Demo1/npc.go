package main

import (
	"math"
	"math/rand"
	"time"
)

// NPC NPC实体
type NPC struct {
	BaseEntity
	Name         string
	NPCTYPE      NPCType
	Level        int
	Health       int
	MaxHealth    int
	AIState      AIState
	Behavior     NPCBehavior
	Path         []Point         // 移动路径
	CurrentPathIndex int
	LastDecision time.Time
	DecisionInterval time.Duration
	HomePosition Point          // 家位置
	WorkPosition Point          // 工作位置
	TargetEntity Entity         // 目标实体
	Inventory    []*Item
}

// NPCType NPC类型
type NPCType int

const (
	NPCVillager NPCType = iota
	NPCGuard
	NPCTrader
	NPCQuestGiver
	NPCTrainer
)

// AIState AI状态
type AIState int

const (
	AIIdle AIState = iota
	AIMoving
	AIWorking
	AIResting
	AIFighting
	AIFleeing
	AITrading
)

// NPCBehavior NPC行为接口
type NPCBehavior interface {
	Update(npc *NPC, deltaTime time.Duration)
	OnPlayerInteract(npc *NPC, player *Player)
	OnDamaged(npc *NPC, attacker Entity, damage int)
	OnDeath(npc *NPC)
}

// NewNPC 创建新NPC
func NewNPC(id, name string, npcType NPCType, position Point) *NPC {
	npc := &NPC{
		BaseEntity: BaseEntity{
			ID:       id,
			Position: position,
			Type:     EntityNPC,
			Alive:    true,
		},
		Name:             name,
		NPCTYPE:          npcType,
		Level:            1,
		Health:           100,
		MaxHealth:        100,
		AIState:          AIIdle,
		Path:             make([]Point, 0),
		CurrentPathIndex: 0,
		LastDecision:     time.Now(),
		DecisionInterval: time.Second * 5, // 5秒做一次决策
		HomePosition:     position,
		WorkPosition:     position,
		Inventory:        make([]*Item, 0),
	}

	// 设置行为
	npc.setBehavior(npcType)

	return npc
}

// setBehavior 根据NPC类型设置行为
func (n *NPC) setBehavior(npcType NPCType) {
	switch npcType {
	case NPCVillager:
		n.Behavior = &VillagerBehavior{}
	case NPCGuard:
		n.Behavior = &GuardBehavior{}
	case NPCTrader:
		n.Behavior = &TraderBehavior{}
	case NPCQuestGiver:
		n.Behavior = &QuestGiverBehavior{}
	case NPCTrainer:
		n.Behavior = &TrainerBehavior{}
	default:
		n.Behavior = &VillagerBehavior{} // 默认村民行为
	}
}

// Update 更新NPC状态
func (n *NPC) Update(deltaTime time.Duration) {
	if !n.Alive {
		return
	}

	// AI行为更新
	if n.Behavior != nil {
		n.Behavior.Update(n, deltaTime)
	}

	// 路径移动
	n.updateMovement(deltaTime)

	// 决策更新
	if time.Since(n.LastDecision) >= n.DecisionInterval {
		n.makeDecision()
		n.LastDecision = time.Now()
	}
}

// updateMovement 更新移动
func (n *NPC) updateMovement(deltaTime time.Duration) {
	if n.AIState == AIMoving && len(n.Path) > 0 && n.CurrentPathIndex < len(n.Path) {
		targetPos := n.Path[n.CurrentPathIndex]

		// 计算移动方向
		dx := float64(targetPos.X - n.Position.X)
		dy := float64(targetPos.Y - n.Position.Y)
		distance := math.Sqrt(dx*dx + dy*dy)

		if distance < 1.0 { // 到达目标点
			n.Position = targetPos
			n.CurrentPathIndex++

			if n.CurrentPathIndex >= len(n.Path) {
				// 到达终点
				n.AIState = AIIdle
				n.Path = nil
				n.CurrentPathIndex = 0
			}
		} else {
			// 向目标移动
			speed := 2.0 // 格子/秒
			moveDistance := speed * deltaTime.Seconds()

			if moveDistance >= distance {
				// 直接到达目标
				n.Position = targetPos
				n.CurrentPathIndex++
			} else {
				// 部分移动
				ratio := moveDistance / distance
				n.Position.X += int(dx * ratio)
				n.Position.Y += int(dy * ratio)
			}
		}
	}
}

// makeDecision 做出决策
func (n *NPC) makeDecision() {
	switch n.NPCTYPE {
	case NPCVillager:
		n.villagerDecision()
	case NPCGuard:
		n.guardDecision()
	case NPCTrader:
		n.traderDecision()
	}
}

// villagerDecision 村民决策
func (n *NPC) villagerDecision() {
	if n.AIState == AIIdle {
		// 随机选择行为
		actions := []AIState{AIMoving, AIWorking, AIResting}
		n.AIState = actions[rand.Intn(len(actions))]

		switch n.AIState {
		case AIMoving:
			// 随机移动到附近位置
			targetX := n.Position.X + rand.Intn(21) - 10 // -10 到 10
			targetY := n.Position.Y + rand.Intn(21) - 10
			n.setPathTo(Point{X: targetX, Y: targetY})

		case AIWorking:
			// 移动到工作位置
			if n.Position != n.WorkPosition {
				n.setPathTo(n.WorkPosition)
			}

		case AIResting:
			// 移动到家
			if n.Position != n.HomePosition {
				n.setPathTo(n.HomePosition)
			}
		}
	}
}

// guardDecision 守卫决策
func (n *NPC) guardDecision() {
	// 守卫通常在固定位置巡逻
	if n.AIState == AIIdle {
		// 检查是否有威胁
		// 这里可以检查周围是否有敌人

		// 如果没有威胁，开始巡逻
		patrolPoints := []Point{
			{X: n.HomePosition.X - 5, Y: n.HomePosition.Y},
			{X: n.HomePosition.X + 5, Y: n.HomePosition.Y},
			{X: n.HomePosition.X, Y: n.HomePosition.Y - 5},
			{X: n.HomePosition.X, Y: n.HomePosition.Y + 5},
		}

		targetIndex := rand.Intn(len(patrolPoints))
		n.setPathTo(patrolPoints[targetIndex])
		n.AIState = AIMoving
	}
}

// traderDecision 商人决策
func (n *NPC) traderDecision() {
	if n.AIState == AIIdle {
		// 商人会在城镇间移动
		// 这里可以实现更复杂的商人AI
		n.AIState = AIResting
	}
}

// setPathTo 设置移动路径到目标点
func (n *NPC) setPathTo(target Point) {
	// 简化路径查找（直线路径）
	// 实际游戏中应该使用A*算法或导航网格
	n.Path = []Point{target}
	n.CurrentPathIndex = 0
	n.AIState = AIMoving
}

// TakeDamage 受到伤害
func (n *NPC) TakeDamage(damage int) int {
	actualDamage := damage
	if actualDamage > n.Health {
		actualDamage = n.Health
	}

	n.Health -= actualDamage

	if n.Health <= 0 {
		n.Health = 0
		n.Alive = false
		if n.Behavior != nil {
			n.Behavior.OnDeath(n)
		}
	} else {
		// 受伤时可能进入战斗或逃跑状态
		if n.AIState != AIFighting && n.AIState != AIFleeing {
			if rand.Float32() < 0.7 { // 70%概率进入战斗
				n.AIState = AIFighting
			} else {
				n.AIState = AIFleeing
				// 逃跑到安全位置
				n.setPathTo(n.HomePosition)
			}
		}

		if n.Behavior != nil {
			n.Behavior.OnDamaged(n, n.TargetEntity, actualDamage)
		}
	}

	return actualDamage
}

// InteractWithPlayer 与玩家交互
func (n *NPC) InteractWithPlayer(player *Player) {
	if n.Behavior != nil {
		n.Behavior.OnPlayerInteract(n, player)
	}
}

// GetName 获取NPC名称
func (n *NPC) GetName() string {
	return n.Name
}

// GetNPCType 获取NPC类型
func (n *NPC) GetNPCType() NPCType {
	return n.NPCTYPE
}

// GetAIState 获取AI状态
func (n *NPC) GetAIState() AIState {
	return n.AIState
}

// SetHomePosition 设置家位置
func (n *NPC) SetHomePosition(pos Point) {
	n.HomePosition = pos
}

// SetWorkPosition 设置工作位置
func (n *NPC) SetWorkPosition(pos Point) {
	n.WorkPosition = pos
}

// AddItem 添加物品
func (n *NPC) AddItem(item *Item) bool {
	n.Inventory = append(n.Inventory, item)
	return true
}

// GetInventory 获取物品列表
func (n *NPC) GetInventory() []*Item {
	inventory := make([]*Item, len(n.Inventory))
	copy(inventory, n.Inventory)
	return inventory
}

// GetNPCStats 获取NPC统计信息
func (n *NPC) GetNPCStats() map[string]interface{} {
	return map[string]interface{}{
		"id":           n.ID,
		"name":         n.Name,
		"type":         n.NPCTYPE,
		"level":        n.Level,
		"health":       n.Health,
		"max_health":   n.MaxHealth,
		"position":     n.Position,
		"ai_state":     n.AIState,
		"inventory_size": len(n.Inventory),
		"home_position": n.HomePosition,
		"work_position": n.WorkPosition,
		"is_alive":     n.Alive,
	}
}

// 行为实现

// VillagerBehavior 村民行为
type VillagerBehavior struct{}

func (vb *VillagerBehavior) Update(npc *NPC, deltaTime time.Duration) {
	// 村民特有的更新逻辑
	// 可以在这里处理工作进度、休息恢复等
}

func (vb *VillagerBehavior) OnPlayerInteract(npc *NPC, player *Player) {
	// 村民与玩家对话
	// 这里可以触发对话、任务、交易等
}

func (vb *VillagerBehavior) OnDamaged(npc *NPC, attacker Entity, damage int) {
	// 村民受伤反应
	// 可能呼救或逃跑
}

func (vb *VillagerBehavior) OnDeath(npc *NPC) {
	// 村民死亡处理
	// 可能触发事件或任务失败
}

// GuardBehavior 守卫行为
type GuardBehavior struct{}

func (gb *GuardBehavior) Update(npc *NPC, deltaTime time.Duration) {
	// 守卫特有逻辑：检查威胁、巡逻等
}

func (gb *GuardBehavior) OnPlayerInteract(npc *NPC, player *Player) {
	// 守卫对话：提供保护、报告情况等
}

func (gb *GuardBehavior) OnDamaged(npc *NPC, attacker Entity, damage int) {
	// 守卫反击
	npc.TargetEntity = attacker
	npc.AIState = AIFighting
}

func (gb *GuardBehavior) OnDeath(npc *NPC) {
	// 守卫死亡：可能触发警报
}

// TraderBehavior 商人行为
type TraderBehavior struct{}

func (tb *TraderBehavior) Update(npc *NPC, deltaTime time.Duration) {
	// 商人特有逻辑：调整价格、补充货物等
}

func (tb *TraderBehavior) OnPlayerInteract(npc *NPC, player *Player) {
	// 开启交易界面
}

func (tb *TraderBehavior) OnDamaged(npc *NPC, attacker Entity, damage int) {
	// 商人逃跑
	npc.AIState = AIFleeing
	npc.setPathTo(npc.HomePosition)
}

func (tb *TraderBehavior) OnDeath(npc *NPC) {
	// 商人死亡：货物掉落
}

// QuestGiverBehavior 任务发布者行为
type QuestGiverBehavior struct{}

func (qgb *QuestGiverBehavior) Update(npc *NPC, deltaTime time.Duration) {
	// 检查任务进度、刷新任务等
}

func (qgb *QuestGiverBehavior) OnPlayerInteract(npc *NPC, player *Player) {
	// 显示可用任务、提交任务等
}

func (qgb *QuestGiverBehavior) OnDamaged(npc *NPC, attacker Entity, damage int) {
	// 任务NPC不会战斗，直接逃跑
	npc.AIState = AIFleeing
}

func (qgb *QuestGiverBehavior) OnDeath(npc *NPC) {
	// 任务失败
}

// TrainerBehavior 训练师行为
type TrainerBehavior struct{}

func (trb *TrainerBehavior) Update(npc *NPC, deltaTime time.Duration) {
	// 训练师逻辑
}

func (trb *TrainerBehavior) OnPlayerInteract(npc *NPC, player *Player) {
	// 技能训练、升级等
}

func (trb *TrainerBehavior) OnDamaged(npc *NPC, attacker Entity, damage int) {
	// 训练师逃跑
	npc.AIState = AIFleeing
}

func (trb *TrainerBehavior) OnDeath(npc *NPC) {
	// 训练师死亡
}
