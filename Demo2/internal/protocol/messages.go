package protocol

import (
	"time"
)

// Point represents a 2D position
type Point struct {
	X, Y int
}

// Rectangle represents a rectangular area
type Rectangle struct {
	X, Y, Width, Height int
}

// EntityType represents the type of entity
type EntityType string

const (
	EntityPlayer EntityType = "player"
	EntityNPC    EntityType = "npc"
)

// GridState represents the state of a grid cell
type GridState struct {
	Position   Point
	Terrain    string
	Resource   *Resource
	Building   *Building
	Occupants  []string // entity IDs
	LastUpdate time.Time
	Version    int64
}

// Resource represents a resource on a grid
type Resource struct {
	Type    string
	Amount  int
	Quality int
}

// Building represents a building on a grid
type Building struct {
	Type      string
	Level     int
	OwnerID   string
	Health    int
	MaxHealth int
}

// Subscriber represents a subscriber to AOI updates
type Subscriber struct {
	PlayerID   string
	AOIManager interface{} // *actor.PID in practice
	ViewRect   Rectangle
	LastUpdate time.Time
}

// BaseEvent represents the base structure for events
type BaseEvent struct {
	EventType string
	Timestamp time.Time
	SourceID  string
	EventData interface{}
}

// GetEventType returns the event type
func (e *BaseEvent) GetEventType() string {
	return e.EventType
}

// GetTimestamp returns the timestamp
func (e *BaseEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

// GetSourceID returns the source ID
func (e *BaseEvent) GetSourceID() string {
	return e.SourceID
}

// EntityMoveEvent represents an entity movement event
type EntityMoveEvent struct {
	BaseEvent
	OpID       string
	EntityID   string
	FromPos    Point
	ToPos      Point
	EntityType EntityType
}

// PlayerJoinEvent represents a player joining event
type PlayerJoinEvent struct {
	BaseEvent
	PlayerID   string
	PlayerName string
	Position   Point
	OpID       string
}

// PlayerLeaveEvent represents a player leaving event
type PlayerLeaveEvent struct {
	BaseEvent
	PlayerID string
	OpID     string
	Reason   string
}

// GridChangeEvent represents a grid state change event
type GridChangeEvent struct {
	BaseEvent
	OpID     string
	GridPos  Point
	OldState *GridState
	NewState *GridState
}

// InteractEvent represents an interaction event (e.g., player with NPC)
type InteractEvent struct {
	BaseEvent
	OpID        string
	InitiatorID string
	TargetID    string
	Action      string
	Position    Point
}

// AOIUpdate represents an AOI update message
type AOIUpdate struct {
	PlayerID        string
	VisibleEntities []VisibleEntity
	VisibleGrids    []GridState
	RemovedEntities []string
	RemovedGrids    []Point
}

// VisibleEntity represents a visible entity in AOI
type VisibleEntity struct {
	ID       string
	Type     EntityType
	Position Point
	State    interface{} // entity-specific state
}

// PersistRequest represents a persistence request
type PersistRequest struct {
	PartitionID string
	Snapshot    []byte
	EntityID    string
	EntityData  []byte
	EventLog    []byte
	OpID        string
}

// PersistResult represents the result of a persistence operation
type PersistResult struct {
	OpID    string
	Success bool
	Error   string
}

// MoveCommand represents a player move command
type MoveCommand struct {
	PlayerID string
	ToPos    Point
	OpID     string
}

// InteractCommand represents a player interaction command
type InteractCommand struct {
	PlayerID string
	TargetID string
	Action   string
	OpID     string
}
