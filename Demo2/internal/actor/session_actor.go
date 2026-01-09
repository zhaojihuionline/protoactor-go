package actor

import (
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"demo/internal/protocol"
)

// SessionActor represents a client session
type SessionActor struct {
	playerID      string
	playerPID     *actor.PID
	eventRouterPID *actor.PID
	lastCommand   time.Time
	rateLimit     time.Duration // Min interval between commands
	outboundCh    chan *protocol.AOIUpdate // Simulated network outbound
}

// NewSessionActor creates a new SessionActor
func NewSessionActor(playerID string, playerPID, eventRouterPID *actor.PID) actor.Actor {
	return &SessionActor{
		playerID:       playerID,
		playerPID:      playerPID,
		eventRouterPID: eventRouterPID,
		rateLimit:      time.Millisecond * 100, // 10 commands per second
		outboundCh:     make(chan *protocol.AOIUpdate, 100),
	}
}

// Receive handles incoming messages
func (sa *SessionActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *protocol.MoveCommand:
		sa.handleMoveCommand(ctx, msg)
	case *protocol.InteractCommand:
		sa.handleInteractCommand(ctx, msg)
	case *protocol.AOIUpdate:
		sa.handleAOIUpdate(msg)
	default:
		log.Printf("SessionActor for %s received unknown message: %T", sa.playerID, msg)
	}
}

// handleMoveCommand processes move commands with rate limiting
func (sa *SessionActor) handleMoveCommand(ctx actor.Context, cmd *protocol.MoveCommand) {
	if !sa.checkRateLimit() {
		log.Printf("Rate limit exceeded for player %s", sa.playerID)
		return
	}

	// Forward to player actor
	ctx.Send(sa.playerPID, cmd)
}

// handleInteractCommand processes interaction commands
func (sa *SessionActor) handleInteractCommand(ctx actor.Context, cmd *protocol.InteractCommand) {
	if !sa.checkRateLimit() {
		log.Printf("Rate limit exceeded for player %s", sa.playerID)
		return
	}

	// Forward to player actor
	ctx.Send(sa.playerPID, cmd)
}

// handleAOIUpdate processes AOI updates (from partitions)
func (sa *SessionActor) handleAOIUpdate(update *protocol.AOIUpdate) {
	// Simulate sending to client
	select {
	case sa.outboundCh <- update:
		log.Printf("AOI update sent to player %s: %d visible entities", sa.playerID, len(update.VisibleEntities))
	default:
		log.Printf("Outbound channel full for player %s", sa.playerID)
	}
}

// checkRateLimit checks if command is allowed based on rate limit
func (sa *SessionActor) checkRateLimit() bool {
	now := time.Now()
	if now.Sub(sa.lastCommand) < sa.rateLimit {
		return false
	}
	sa.lastCommand = now
	return true
}

// GetOutboundChannel returns the outbound channel for testing
func (sa *SessionActor) GetOutboundChannel() <-chan *protocol.AOIUpdate {
	return sa.outboundCh
}

// SimulateClientInput simulates receiving commands from client
func (sa *SessionActor) SimulateClientInput(ctx actor.Context, cmd interface{}) {
	ctx.Send(ctx.Self(), cmd)
}
