package actor

import (
	"log"
	"math"

	"github.com/asynkron/protoactor-go/actor"
	"demo/internal/protocol"
)

// AOIActor computes Area of Interest updates
type AOIActor struct {
	subscribers map[string]*protocol.Subscriber
	viewRadius  int // Default view radius
}

// NewAOIActor creates a new AOIActor
func NewAOIActor(viewRadius int) actor.Actor {
	return &AOIActor{
		subscribers: make(map[string]*protocol.Subscriber),
		viewRadius:  viewRadius,
	}
}

// Receive handles AOI computation requests
func (aoi *AOIActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *AOIComputeRequest:
		aoi.computeAOI(ctx, msg)
	default:
		log.Printf("AOIActor received unknown message: %T", msg)
	}
}

// AOIComputeRequest represents a request to compute AOI
type AOIComputeRequest struct {
	PlayerID  string
	CenterPos protocol.Point
	Entities  map[string]interface{} // entityID -> entity data
	Grids     map[protocol.Point]*protocol.GridState
}

// computeAOI computes visible entities and grids for a player
func (aoi *AOIActor) computeAOI(ctx actor.Context, req *AOIComputeRequest) {
	visibleEntities := []protocol.VisibleEntity{}
	visibleGrids := []*protocol.GridState{}
	removedEntities := []string{}
	removedGrids := []protocol.Point{}

	// Compute visible area
	minX := req.CenterPos.X - aoi.viewRadius
	maxX := req.CenterPos.X + aoi.viewRadius
	minY := req.CenterPos.Y - aoi.viewRadius
	maxY := req.CenterPos.Y + aoi.viewRadius

	// Check entities
	for entityID, entityData := range req.Entities {
		if entity, ok := entityData.(map[string]interface{}); ok {
			if pos, exists := entity["position"].(protocol.Point); exists {
				distance := aoi.distance(req.CenterPos, pos)
				if distance <= float64(aoi.viewRadius) {
					visibleEntities = append(visibleEntities, protocol.VisibleEntity{
						ID:       entityID,
						Position: pos,
						State:    entity,
					})
				}
			}
		}
	}

	// Check grids
	for pos, grid := range req.Grids {
		if pos.X >= minX && pos.X <= maxX && pos.Y >= minY && pos.Y <= maxY {
			visibleGrids = append(visibleGrids, grid)
		}
	}

	// Create AOI update
	update := &protocol.AOIUpdate{
		PlayerID:         req.PlayerID,
		VisibleEntities:  visibleEntities,
		VisibleGrids:     visibleGrids,
		RemovedEntities:  removedEntities, // Simplified
		RemovedGrids:     removedGrids,    // Simplified
	}

	// Send back to requester (usually the partition)
	ctx.Respond(update)
}

// distance calculates Euclidean distance
func (aoi *AOIActor) distance(p1, p2 protocol.Point) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

// AddSubscriber adds a subscriber
func (aoi *AOIActor) AddSubscriber(playerID string, sub *protocol.Subscriber) {
	aoi.subscribers[playerID] = sub
}

// RemoveSubscriber removes a subscriber
func (aoi *AOIActor) RemoveSubscriber(playerID string) {
	delete(aoi.subscribers, playerID)
}
