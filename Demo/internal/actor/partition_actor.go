package actor

import (
	"encoding/json"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"demo/internal/protocol"
)

// PartitionActor manages a 100x100 grid partition
type PartitionActor struct {
	ID            string
	Bounds        protocol.Rectangle
	GridStates    map[protocol.Point]*protocol.GridState
	Entities      map[string]interface{} // entity ID -> entity data
	Subscribers   map[string]*protocol.Subscriber
	StateVersion  int64
	RedisWriter   *actor.PID
}

// NewPartitionActor creates a new PartitionActor
func NewPartitionActor(id string, bounds protocol.Rectangle, redisWriter *actor.PID) actor.Actor {
	p := &PartitionActor{
		ID:           id,
		Bounds:       bounds,
		GridStates:   make(map[protocol.Point]*protocol.GridState),
		Entities:     make(map[string]interface{}),
		Subscribers:  make(map[string]*protocol.Subscriber),
		StateVersion: 1,
		RedisWriter:  redisWriter,
	}

	// Initialize grid states (100x100)
	p.initializeGridStates()

	return p
}

// initializeGridStates initializes the 100x100 grid
func (p *PartitionActor) initializeGridStates() {
	for x := p.Bounds.X; x < p.Bounds.X+p.Bounds.Width; x++ {
		for y := p.Bounds.Y; y < p.Bounds.Y+p.Bounds.Height; y++ {
			pos := protocol.Point{X: x, Y: y}
			p.GridStates[pos] = &protocol.GridState{
				Position:   pos,
				Terrain:    "grass", // default
				Resource:   nil,
				Building:   nil,
				Occupants:  []string{},
				LastUpdate: time.Now(),
				Version:    1,
			}
		}
	}
}

// Receive handles incoming messages
func (p *PartitionActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *protocol.EntityMoveEvent:
		p.handleEntityMove(msg)
	case *protocol.PlayerJoinEvent:
		p.handlePlayerJoin(msg)
	case *protocol.PlayerLeaveEvent:
		p.handlePlayerLeave(msg)
	case *protocol.GridChangeEvent:
		p.handleGridChange(msg)
	case *protocol.InteractEvent:
		p.handleInteract(msg)
	default:
		log.Printf("PartitionActor %s received unknown message: %T", p.ID, msg)
	}
}

// handleEntityMove processes entity movement
func (p *PartitionActor) handleEntityMove(msg *protocol.EntityMoveEvent) {
	// Update entity position
	if entity, exists := p.Entities[msg.EntityID]; exists {
		// Simplified: just update position
		// In real implementation, update entity data
		log.Printf("Entity %s moved from (%d,%d) to (%d,%d) in partition %s",
			msg.EntityID, msg.FromPos.X, msg.FromPos.Y, msg.ToPos.X, msg.ToPos.Y, p.ID)
	}

	// Compute AOI diff and notify subscribers
	p.computeAOIAndNotify()

	// Trigger persistence
	p.persistState(msg.OpID)
}

// handlePlayerJoin processes player joining
func (p *PartitionActor) handlePlayerJoin(msg *protocol.PlayerJoinEvent) {
	// Add entity
	p.Entities[msg.PlayerID] = map[string]interface{}{
		"id":       msg.PlayerID,
		"name":     msg.PlayerName,
		"position": msg.Position,
		"type":     protocol.EntityPlayer,
	}

	log.Printf("Player %s joined partition %s at (%d,%d)",
		msg.PlayerID, p.ID, msg.Position.X, msg.Position.Y)

	// Compute AOI and notify
	p.computeAOIAndNotify()

	// Persist
	p.persistState(msg.OpID)
}

// handlePlayerLeave processes player leaving
func (p *PartitionActor) handlePlayerLeave(msg *protocol.PlayerLeaveEvent) {
	delete(p.Entities, msg.PlayerID)
	log.Printf("Player %s left partition %s", msg.PlayerID, p.ID)

	// Compute AOI and notify
	p.computeAOIAndNotify()

	// Persist
	p.persistState(msg.OpID)
}

// handleGridChange processes grid state changes
func (p *PartitionActor) handleGridChange(msg *protocol.GridChangeEvent) {
	if grid, exists := p.GridStates[msg.GridPos]; exists {
		*grid = *msg.NewState
		grid.LastUpdate = time.Now()
		grid.Version = p.StateVersion + 1
		p.StateVersion = grid.Version

		log.Printf("Grid (%d,%d) changed in partition %s", msg.GridPos.X, msg.GridPos.Y, p.ID)
	}

	// Compute AOI and notify
	p.computeAOIAndNotify()

	// Persist
	p.persistState(msg.OpID)
}

// handleInteract processes interactions
func (p *PartitionActor) handleInteract(msg *protocol.InteractEvent) {
	log.Printf("Interaction %s -> %s in partition %s", msg.InitiatorID, msg.TargetID, p.ID)

	// Simplified: just log and persist
	p.persistState(msg.OpID)
}

// computeAOIAndNotify computes AOI differences and notifies subscribers
func (p *PartitionActor) computeAOIAndNotify() {
	// Simplified AOI computation
	// In real implementation, calculate visible entities/grids for each subscriber
	for playerID, sub := range p.Subscribers {
		update := &protocol.AOIUpdate{
			PlayerID: playerID,
			// Populate VisibleEntities, etc.
		}
		// Send to subscriber's AOI manager (simplified)
		log.Printf("Sending AOI update to player %s", playerID)
	}
}

// persistState sends persistence request
func (p *PartitionActor) persistState(opID string) {
	snapshot, _ := json.Marshal(p)
	req := &protocol.PersistRequest{
		PartitionID: p.ID,
		Snapshot:    snapshot,
		OpID:        opID,
	}
	// Send to RedisWriterActor asynchronously
	actor.NewPID("127.0.0.1:0", "redis_writer").Request(req, actor.NewFuture(5*time.Second)) // Simplified PID
}
