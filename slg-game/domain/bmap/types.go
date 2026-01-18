package bmap

/*
	公共类型定义
*/

import "time"

type LayerNumber int32

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
	X float64
	Y float64
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
	Entities  map[int64]*Entity
	Grids     map[int64]*Grid
	Timestamp time.Time
}
