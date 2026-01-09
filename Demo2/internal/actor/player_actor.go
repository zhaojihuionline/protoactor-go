package actor

import (
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"demo/internal/protocol"
)

// Player represents a player entity
type Player struct {
	ID         string
	Name       string
	Position   protocol.Point
	Level      int
	Health     int
	MaxHealth  int
	LastActive time.Time
}

// PlayerActor manages player state and commands
type PlayerActor struct {
	player         *Player
	eventRouterPID *actor.PID
	partitionPID   *actor.PID
	lastActive     time.Time
	passivateAfter time.Duration
}

// NewPlayerActor creates a new PlayerActor
func NewPlayerActor(playerID, playerName string, startPos protocol.Point, eventRouterPID *actor.PID) actor.Actor {
	player := &Player{
		ID:         playerID,
		Name:       playerName,
		Position:   startPos,
		Level:      1,
		Health:     100,
		MaxHealth:  100,
		LastActive: time.Now(),
	}

	return &PlayerActor{
		player:         player,
		eventRouterPID: eventRouterPID,
		lastActive:     time.Now(),
		passivateAfter: time.Minute * 30, // Passivate after 30 minutes of inactivity
	}
}

// Receive handles incoming messages
func (pa *PlayerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *protocol.MoveCommand:
		pa.handleMoveCommand(ctx, msg)
	case *protocol.InteractCommand:
		pa.handleInteractCommand(ctx, msg)
	case *actor.SystemMessage:
		pa.handleSystemMessage(ctx, msg)
	default:
		log.Printf("PlayerActor %s received unknown message: %T", pa.player.ID, msg)
	}
}

// handleMoveCommand processes move commands
func (pa *PlayerActor) handleMoveCommand(ctx actor.Context, cmd *protocol.MoveCommand) {
	fromPos := pa.player.Position
	pa.player.Position = cmd.ToPos
	pa.lastActive = time.Now()

	// Create move event
	event := &protocol.EntityMoveEvent{
		BaseEvent: protocol.BaseEvent{
			EventType: "entity_move",
			Timestamp: time.Now(),
			SourceID:  pa.player.ID,
		},
		OpID:       cmd.OpID,
		EntityID:   pa.player.ID,
		FromPos:    fromPos,
		ToPos:      cmd.ToPos,
		EntityType: protocol.EntityPlayer,
	}

	// Send to event router
	ctx.Send(pa.eventRouterPID, event)

	log.Printf("Player %s moved from (%d,%d) to (%d,%d)", pa.player.ID, fromPos.X, fromPos.Y, cmd.ToPos.X, cmd.ToPos.Y)
}

// handleInteractCommand processes interaction commands
func (pa *PlayerActor) handleInteractCommand(ctx actor.Context, cmd *protocol.InteractCommand) {
	pa.lastActive = time.Now()

	// Create interact event
	event := &protocol.InteractEvent{
		BaseEvent: protocol.BaseEvent{
			EventType: "interact",
			Timestamp: time.Now(),
			SourceID:  pa.player.ID,
		},
		OpID:        cmd.OpID,
		InitiatorID: pa.player.ID,
		TargetID:    cmd.TargetID,
		Action:      cmd.Action,
		Position:    pa.player.Position,
	}

	// Send to event router
	ctx.Send(pa.eventRouterPID, event)

	log.Printf("Player %s interacted with %s", pa.player.ID, cmd.TargetID)
}

// handleSystemMessage handles system messages like passivation
func (pa *PlayerActor) handleSystemMessage(ctx actor.Context, msg *actor.SystemMessage) {
	switch msg.Header {
	case "passivate":
		// Check if inactive
		if time.Since(pa.lastActive) > pa.passivateAfter {
			log.Printf("Passivating player actor %s", pa.player.ID)
			// In practice, save state to Redis and stop
			ctx.Stop(ctx.Self())
		}
	}
}

// GetPlayer returns the player data
func (pa *PlayerActor) GetPlayer() *Player {
	return pa.player
}

// UpdateActivity updates the last active time
func (pa *PlayerActor) UpdateActivity() {
	pa.lastActive = time.Now()
}
