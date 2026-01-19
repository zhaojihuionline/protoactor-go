package bmap

/*
	公共类型定义
*/

import (
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// 地图和分区尺寸常量
const (
	MAP_WIDTH        = 1200 // 地图宽度
	MAP_HEIGHT       = 1200 // 地图高度
	PARTITION_WIDTH  = 100  // 分区宽度 (必须能整除MAP_WIDTH)
	PARTITION_HEIGHT = 100  // 分区高度 (必须能整除MAP_HEIGHT)
)

type LayerNumber int32

const LayerCount = 8
const (
	LayerNumber1 LayerNumber = 1
	LayerNumber2 LayerNumber = 2
	LayerNumber3 LayerNumber = 3
	LayerNumber4 LayerNumber = 4
	LayerNumber5 LayerNumber = 5
	LayerNumber6 LayerNumber = 6
	LayerNumber7 LayerNumber = 7
	LayerNumber8 LayerNumber = 8
)

type EntityID int64
type EntityType int32

const (
	EntityTypeTerrain        EntityType = 100
	EntityTypePlayerMainCity EntityType = 101
	EntityTypeFortress       EntityType = 102
	EntityTypeStronghold     EntityType = 103
	EntityTypeMonsterBear    EntityType = 200
	EntityTypeMonsterWolf    EntityType = 201
	EntityTypeOther          EntityType = 300
)

type Position struct {
	X int32
	Y int32
}

type View struct {
	W int32
	H int32
}

type Entity struct {
	ID       int64
	Type     EntityType
	Data     any
	Position *Position
	Version  int64
	GridID   int64
}

type Grid struct {
	ID       int64
	Position *Position
	Version  int64
	EntityID int64
}

type AOICache struct {
	Entities  map[int32]*Entity
	Grids     map[int32]*Grid
	Timestamp time.Time
}

// 消息类型定义

// EnterMap 进入大地图消息
type EnterMap struct {
	Layer  LayerNumber
	Center Position
	View   View
}

// MoveView 视野移动消息
type MoveView struct {
	Layer  LayerNumber
	Center Position
	View   View
}

// LeaveMap 离开大地图消息
type LeaveMap struct {
}

// SubscribePlayer 订阅玩家消息（发送给PartionActor）
type SubscribePlayer struct {
	Layer LayerNumber
	PID   *actor.PID
}

// UnsubscribePlayer 取消订阅玩家消息（发送给PartionActor）
type UnsubscribePlayer struct {
	Layer LayerNumber
	PID   *actor.PID
}
