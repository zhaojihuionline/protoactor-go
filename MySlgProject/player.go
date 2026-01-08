package main

import (
	"time"
)

// Player 玩家实体
type Player struct {
	BaseEntity
	Name         string
	Level        int
	Experience   int
	Health       int
	MaxHealth    int
	Mana         int
	MaxMana      int
	Resources    map[ResourceType]int
	Inventory    []*Item
	Skills       []*Skill
	Achievements []string
	LastActive   time.Time
	AOIManager   *AOIManager
}

// Item 物品
type Item struct {
	ID          string
	Name        string
	Type        ItemType
	Quantity    int
	Quality     int
	Attributes  map[string]interface{}
}

// ItemType 物品类型
type ItemType int

const (
	ItemWeapon ItemType = iota
	ItemArmor
	ItemConsumable
	ItemMaterial
	ItemQuest
)

// Skill 技能
type Skill struct {
	ID          string
	Name        string
	Level       int
	MaxLevel    int
	Cooldown    time.Duration
	LastUsed    time.Time
}

// NewPlayer 创建新玩家
func NewPlayer(id, name string, startPos Point) *Player {
	player := &Player{
		BaseEntity: BaseEntity{
			ID:       id,
			Position: startPos,
			Type:     EntityPlayer,
			Alive:    true,
		},
		Name:       name,
		Level:      1,
		Experience: 0,
		Health:     100,
		MaxHealth:  100,
		Mana:       50,
		MaxMana:    50,
		Resources: map[ResourceType]int{
			ResourceWood:  100,
			ResourceStone: 50,
			ResourceGold:  200,
			ResourceFood:  150,
		},
		Inventory:    make([]*Item, 0),
		Skills:       make([]*Skill, 0),
		Achievements: make([]string, 0),
		LastActive:   time.Now(),
	}

	// 初始化基础技能
	player.initializeSkills()

	return player
}

// initializeSkills 初始化玩家技能
func (p *Player) initializeSkills() {
	p.Skills = []*Skill{
		{
			ID:       "basic_attack",
			Name:     "基础攻击",
			Level:    1,
			MaxLevel: 10,
			Cooldown: time.Second * 1,
		},
		{
			ID:       "gather",
			Name:     "采集",
			Level:    1,
			MaxLevel: 5,
			Cooldown: time.Second * 2,
		},
		{
			ID:       "build",
			Name:     "建造",
			Level:    1,
			MaxLevel: 3,
			Cooldown: time.Second * 5,
		},
	}
}

// GetName 获取玩家名称
func (p *Player) GetName() string {
	return p.Name
}

// GetLevel 获取玩家等级
func (p *Player) GetLevel() int {
	return p.Level
}

// AddExperience 增加经验值
func (p *Player) AddExperience(amount int) {
	p.Experience += amount

	// 检查是否升级
	requiredExp := p.Level * 100 // 简化升级经验计算
	for p.Experience >= requiredExp && p.Level < 100 {
		p.Experience -= requiredExp
		p.Level++
		p.onLevelUp()
		requiredExp = p.Level * 100
	}
}

// onLevelUp 升级处理
func (p *Player) onLevelUp() {
	// 增加属性
	p.MaxHealth += 10
	p.MaxMana += 5
	p.Health = p.MaxHealth // 满血
	p.Mana = p.MaxMana     // 满蓝

	// 解锁新技能或升级现有技能
	p.unlockNewSkills()
}

// unlockNewSkills 解锁新技能
func (p *Player) unlockNewSkills() {
	switch p.Level {
	case 5:
		p.Skills = append(p.Skills, &Skill{
			ID:       "fireball",
			Name:     "火球术",
			Level:    1,
			MaxLevel: 10,
			Cooldown: time.Second * 3,
		})
	case 10:
		p.Skills = append(p.Skills, &Skill{
			ID:       "teleport",
			Name:     "传送",
			Level:    1,
			MaxLevel: 5,
			Cooldown: time.Second * 10,
		})
	}
}

// TakeDamage 受到伤害
func (p *Player) TakeDamage(damage int) int {
	actualDamage := damage
	if actualDamage > p.Health {
		actualDamage = p.Health
	}

	p.Health -= actualDamage

	if p.Health <= 0 {
		p.Health = 0
		p.Alive = false
	}

	return actualDamage
}

// Heal 治疗
func (p *Player) Heal(amount int) int {
	actualHeal := amount
	if p.Health + actualHeal > p.MaxHealth {
		actualHeal = p.MaxHealth - p.Health
	}

	p.Health += actualHeal
	return actualHeal
}

// AddResource 增加资源
func (p *Player) AddResource(resourceType ResourceType, amount int) {
	if p.Resources == nil {
		p.Resources = make(map[ResourceType]int)
	}
	p.Resources[resourceType] += amount
}

// RemoveResource 移除资源
func (p *Player) RemoveResource(resourceType ResourceType, amount int) bool {
	if current, exists := p.Resources[resourceType]; exists && current >= amount {
		p.Resources[resourceType] -= amount
		return true
	}
	return false
}

// GetResource 获取资源数量
func (p *Player) GetResource(resourceType ResourceType) int {
	if amount, exists := p.Resources[resourceType]; exists {
		return amount
	}
	return 0
}

// AddItem 添加物品到背包
func (p *Player) AddItem(item *Item) bool {
	// 检查背包是否已满（简化实现，假设背包容量50）
	if len(p.Inventory) >= 50 {
		return false
	}

	p.Inventory = append(p.Inventory, item)
	return true
}

// RemoveItem 从背包移除物品
func (p *Player) RemoveItem(itemID string, quantity int) bool {
	for i, item := range p.Inventory {
		if item.ID == itemID && item.Quantity >= quantity {
			item.Quantity -= quantity
			if item.Quantity <= 0 {
				// 移除物品
				p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
			}
			return true
		}
	}
	return false
}

// UseSkill 使用技能
func (p *Player) UseSkill(skillID string) bool {
	for _, skill := range p.Skills {
		if skill.ID == skillID {
			// 检查冷却时间
			if time.Since(skill.LastUsed) < skill.Cooldown {
				return false
			}

			// 检查法力值
			manaCost := skill.Level * 5
			if p.Mana < manaCost {
				return false
			}

			// 消耗法力
			p.Mana -= manaCost
			skill.LastUsed = time.Now()

			return true
		}
	}
	return false
}

// GetSkill 获取技能
func (p *Player) GetSkill(skillID string) *Skill {
	for _, skill := range p.Skills {
		if skill.ID == skillID {
			return skill
		}
	}
	return nil
}

// UpgradeSkill 升级技能
func (p *Player) UpgradeSkill(skillID string) bool {
	for _, skill := range p.Skills {
		if skill.ID == skillID && skill.Level < skill.MaxLevel {
			skill.Level++
			return true
		}
	}
	return false
}

// AddAchievement 添加成就
func (p *Player) AddAchievement(achievementID string) {
	for _, achievement := range p.Achievements {
		if achievement == achievementID {
			return // 已获得
		}
	}
	p.Achievements = append(p.Achievements, achievementID)
}

// HasAchievement 检查是否拥有成就
func (p *Player) HasAchievement(achievementID string) bool {
	for _, achievement := range p.Achievements {
		if achievement == achievementID {
			return true
		}
	}
	return false
}

// UpdateActivity 更新活动时间
func (p *Player) UpdateActivity() {
	p.LastActive = time.Now()
}

// IsOnline 检查是否在线（简化实现）
func (p *Player) IsOnline() bool {
	return time.Since(p.LastActive) < time.Minute*30 // 30分钟内活跃算在线
}

// GetPlayerStats 获取玩家统计信息
func (p *Player) GetPlayerStats() map[string]interface{} {
	stats := map[string]interface{}{
		"id":           p.ID,
		"name":         p.Name,
		"level":        p.Level,
		"experience":   p.Experience,
		"health":       p.Health,
		"max_health":   p.MaxHealth,
		"mana":         p.Mana,
		"max_mana":     p.MaxMana,
		"position":     p.Position,
		"resources":    p.Resources,
		"inventory_size": len(p.Inventory),
		"skill_count": len(p.Skills),
		"achievement_count": len(p.Achievements),
		"last_active":  p.LastActive,
		"is_online":    p.IsOnline(),
		"is_alive":     p.Alive,
	}

	return stats
}

// GetInventory 获取背包物品
func (p *Player) GetInventory() []*Item {
	inventory := make([]*Item, len(p.Inventory))
	copy(inventory, p.Inventory)
	return inventory
}

// GetSkills 获取技能列表
func (p *Player) GetSkills() []*Skill {
	skills := make([]*Skill, len(p.Skills))
	copy(skills, skills)
	return skills
}

// GetAchievements 获取成就列表
func (p *Player) GetAchievements() []string {
	achievements := make([]string, len(p.Achievements))
	copy(achievements, p.Achievements)
	return achievements
}

// CalculateCombatPower 计算战斗力
func (p *Player) CalculateCombatPower() int {
	power := p.Level * 10 // 基础战斗力

	// 技能加成
	for _, skill := range p.Skills {
		power += skill.Level * 5
	}

	// 装备加成（简化实现）
	for _, item := range p.Inventory {
		if item.Type == ItemWeapon || item.Type == ItemArmor {
			power += item.Quality * 2
		}
	}

	return power
}

// IsInCombat 检查是否在战斗中（简化实现）
func (p *Player) IsInCombat() bool {
	// 这里可以根据实体的状态、周围敌人等判断
	// 暂时返回false
	return false
}

// GetRespawnPosition 获取重生位置（简化实现）
func (p *Player) GetRespawnPosition() Point {
	// 返回城镇中心或其他安全位置
	return Point{X: 600, Y: 600}
}

// Respawn 重生
func (p *Player) Respawn() {
	if !p.Alive {
		p.Alive = true
		p.Health = p.MaxHealth / 2 // 重生时生命值减半
		p.Position = p.GetRespawnPosition()
	}
}

// Save 保存玩家数据（接口定义）
func (p *Player) Save() error {
	// 这里应该实现数据持久化
	// 暂时返回nil
	return nil
}

// Load 加载玩家数据（接口定义）
func (p *Player) Load() error {
	// 这里应该实现数据加载
	// 暂时返回nil
	return nil
}
