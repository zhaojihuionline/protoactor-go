package main

import (
	"fmt"
	"time"
)

// Point 坐标点
type Point struct {
	X, Y int
}

func (p Point) String() string {
	return fmt.Sprintf("(%d,%d)", p.X, p.Y)
}

// Distance 计算两点间距离
func (p Point) Distance(other Point) float64 {
	dx := float64(p.X - other.X)
	dy := float64(p.Y - other.Y)
	return dx*dx + dy*dy // 简化版，实际可使用math.Sqrt
}

// Add 坐标相加
func (p Point) Add(other Point) Point {
	return Point{X: p.X + other.X, Y: p.Y + other.Y}
}

// Rectangle 矩形区域
type Rectangle struct {
	X, Y          int // 左上角坐标
	Width, Height int
}

func (r Rectangle) String() string {
	return fmt.Sprintf("Rect(%d,%d,%dx%d)", r.X, r.Y, r.Width, r.Height)
}

// Contains 检查点是否在矩形内
func (r Rectangle) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.Width &&
		p.Y >= r.Y && p.Y < r.Y+r.Height
}

// Intersects 检查两个矩形是否相交
func (r Rectangle) Intersects(other Rectangle) bool {
	return !(r.X+r.Width <= other.X || other.X+other.Width <= r.X ||
		r.Y+r.Height <= other.Y || other.Y+other.Height <= r.Y)
}

// GridState 格子状态
type GridState struct {
	Position   Point
	Terrain    TerrainType // 地形类型
	Resource   *Resource   // 资源
	Building   *Building   // 建筑
	Occupants  []*Entity   // 占用者
	LastUpdate time.Time
	Version    int64
}

// TerrainType 地形类型
type TerrainType int

const (
	TerrainGrass TerrainType = iota
	TerrainForest
	TerrainMountain
	TerrainWater
	TerrainDesert
)

// Resource 资源
type Resource struct {
	Type        ResourceType
	Amount      int
	RegenRate   int // 每分钟恢复量
	LastHarvest time.Time
}

// ResourceType 资源类型
type ResourceType int

const (
	ResourceWood ResourceType = iota
	ResourceStone
	ResourceGold
	ResourceFood
)

// Building 建筑
type Building struct {
	ID        string
	Type      BuildingType
	OwnerID   string
	Level     int
	Health    int
	MaxHealth int
	Created   time.Time
}

// BuildingType 建筑类型
type BuildingType int

const (
	BuildingTownHall BuildingType = iota
	BuildingBarracks
	BuildingFarm
	BuildingMine
)

// Entity 游戏实体接口
type Entity interface {
	GetID() string
	GetPosition() Point
	SetPosition(Point)
	GetType() EntityType
	IsAlive() bool
}

// BaseEntity 基础实体实现
type BaseEntity struct {
	ID       string
	Position Point
	Type     EntityType
	Alive    bool
}

func (e *BaseEntity) GetID() string {
	return e.ID
}

func (e *BaseEntity) GetPosition() Point {
	return e.Position
}

func (e *BaseEntity) SetPosition(pos Point) {
	e.Position = pos
}

func (e *BaseEntity) GetType() EntityType {
	return e.Type
}

func (e *BaseEntity) IsAlive() bool {
	return e.Alive
}

// EntityType 实体类型
type EntityType int

const (
	EntityPlayer EntityType = iota
	EntityNPC
	EntityMonster
	EntityBuilding
)

// AOIEvent AOI事件
type AOIEvent struct {
	EventType AOIEventType
	EntityID  string
	Position  Point
	Data      interface{}
	Timestamp time.Time
}

// AOIEventType AOI事件类型
type AOIEventType int

const (
	AOIEntityEnter AOIEventType = iota
	AOIEntityLeave
	AOIEntityMove
	AOIEventTypeStateChange
	AOIEntitySpawn
	AOIEntityDespawn
)

// StateChange 状态变更
type StateChange struct {
	GridPos    Point
	ChangeType ChangeType
	OldState   interface{}
	NewState   interface{}
	Timestamp  time.Time
	Version    int64
}

// ChangeType 变更类型
type ChangeType int

const (
	ChangeTerrain ChangeType = iota
	ChangeResource
	ChangeBuilding
	ChangeOccupant
)

// PartitionMetrics 分区指标
type PartitionMetrics struct {
	PartitionID     string
	PlayerCount     int
	EntityCount     int
	MessageRate     float64 // msg/s
	CpuUsage        float64
	MemoryUsage     float64
	AvgResponseTime time.Duration
	LastUpdate      time.Time
}
