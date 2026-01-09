package actor

import (
	"crypto/rand"
	"fmt"
	"log"

	"github.com/asynkron/protoactor-go/actor"
	"demo/internal/protocol"
	"demo/internal/manager"
)

// EventRouterActor routes events to appropriate PartitionActors
type EventRouterActor struct {
	PartitionManager *manager.PartitionManager
	ProcessedOpIDs   map[string]bool // Simple in-memory dedup (in production, use Redis)
}

// NewEventRouterActor creates a new EventRouterActor
func NewEventRouterActor(pm *manager.PartitionManager) actor.Actor {
	return &EventRouterActor{
		PartitionManager: pm,
		ProcessedOpIDs:   make(map[string]bool),
	}
}

// Receive handles incoming events
func (er *EventRouterActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *protocol.EntityMoveEvent:
		er.routeEntityMove(msg)
	case *protocol.PlayerJoinEvent:
		er.routePlayerJoin(msg)
	case *protocol.PlayerLeaveEvent:
		er.routePlayerLeave(msg)
	case *protocol.GridChangeEvent:
		er.routeGridChange(msg)
	case *protocol.InteractEvent:
		er.routeInteract(msg)
	default:
		log.Printf("EventRouter received unknown message: %T", msg)
	}
}

// routeEntityMove routes entity move event
func (er *EventRouterActor) routeEntityMove(msg *protocol.EntityMoveEvent) {
	if er.isDuplicate(msg.OpID) {
		log.Printf("Duplicate OpID %s, ignoring", msg.OpID)
		return
	}

	partitionID := er.getPartitionID(msg.ToPos)
	if pid, exists := er.PartitionManager.GetPartitionPID(partitionID); exists {
		ctx.Send(pid, msg)
	} else {
		log.Printf("Partition %s not found for position (%d,%d)", partitionID, msg.ToPos.X, msg.ToPos.Y)
	}
}

// routePlayerJoin routes player join event
func (er *EventRouterActor) routePlayerJoin(msg *protocol.PlayerJoinEvent) {
	if er.isDuplicate(msg.OpID) {
		log.Printf("Duplicate OpID %s, ignoring", msg.OpID)
		return
	}

	partitionID := er.getPartitionID(msg.Position)
	if pid, exists := er.PartitionManager.GetPartitionPID(partitionID); exists {
		ctx.Send(pid, msg)
	} else {
		log.Printf("Partition %s not found for position (%d,%d)", partitionID, msg.Position.X, msg.Position.Y)
	}
}

// routePlayerLeave routes player leave event
func (er *EventRouterActor) routePlayerLeave(msg *protocol.PlayerLeaveEvent) {
	if er.isDuplicate(msg.OpID) {
		log.Printf("Duplicate OpID %s, ignoring", msg.OpID)
		return
	}

	// For leave, we need to find which partition the player is in
	// Simplified: assume we can lookup from manager or Redis
	// In practice, maintain player->partition mapping
	partitionID := er.getPartitionForPlayer(msg.PlayerID)
	if pid, exists := er.PartitionManager.GetPartitionPID(partitionID); exists {
		ctx.Send(pid, msg)
	} else {
		log.Printf("Partition %s not found for player %s", partitionID, msg.PlayerID)
	}
}

// routeGridChange routes grid change event
func (er *EventRouterActor) routeGridChange(msg *protocol.GridChangeEvent) {
	if er.isDuplicate(msg.OpID) {
		log.Printf("Duplicate OpID %s, ignoring", msg.OpID)
		return
	}

	partitionID := er.getPartitionID(msg.GridPos)
	if pid, exists := er.PartitionManager.GetPartitionPID(partitionID); exists {
		ctx.Send(pid, msg)
	} else {
		log.Printf("Partition %s not found for grid position (%d,%d)", partitionID, msg.GridPos.X, msg.GridPos.Y)
	}
}

// routeInteract routes interaction event
func (er *EventRouterActor) routeInteract(msg *protocol.InteractEvent) {
	if er.isDuplicate(msg.OpID) {
		log.Printf("Duplicate OpID %s, ignoring", msg.OpID)
		return
	}

	partitionID := er.getPartitionID(msg.Position)
	if pid, exists := er.PartitionManager.GetPartitionPID(partitionID); exists {
		ctx.Send(pid, msg)
	} else {
		log.Printf("Partition %s not found for interaction at (%d,%d)", partitionID, msg.Position.X, msg.Position.Y)
	}
}

// getPartitionID calculates partition ID from position
func (er *EventRouterActor) getPartitionID(pos protocol.Point) string {
	// Assuming 100x100 partitions, 12x12 grid
	partitionSize := 100
	row := pos.Y / partitionSize
	col := pos.X / partitionSize
	return fmt.Sprintf("partition_%d_%d", row, col)
}

// getPartitionForPlayer finds partition for a player (simplified)
func (er *EventRouterActor) getPartitionForPlayer(playerID string) string {
	// In practice, lookup from Redis or in-memory map
	// For demo, assume partition_0_0
	return "partition_0_0"
}

// isDuplicate checks if OpID was already processed
func (er *EventRouterActor) isDuplicate(opID string) bool {
	if opID == "" {
		return false // Allow empty for testing
	}
	if er.ProcessedOpIDs[opID] {
		return true
	}
	er.ProcessedOpIDs[opID] = true
	return false
}

// generateOpID generates a new operation ID
func generateOpID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("%x", bytes)
}
