package manager

import (
	"fmt"
	"sync"

	"demo/internal/actor"
	"demo/internal/protocol"

	"github.com/asynkron/protoactor-go/actor"
)

// PartitionManager manages all partition actors
type PartitionManager struct {
	system        *actor.ActorSystem
	partitions    map[string]*actor.PID // partitionID -> PID
	redisWriter   *actor.PID
	partitionSize int // 100
	gridSize      int // 12
	mutex         sync.RWMutex
}

// NewPartitionManager creates a new PartitionManager
func NewPartitionManager(system *actor.ActorSystem, redisWriter *actor.PID) *PartitionManager {
	return &PartitionManager{
		system:        system,
		partitions:    make(map[string]*actor.PID),
		redisWriter:   redisWriter,
		partitionSize: 100,
		gridSize:      12,
	}
}

// StartAllPartitions creates and starts all 144 partition actors
func (pm *PartitionManager) StartAllPartitions() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for row := 0; row < pm.gridSize; row++ {
		for col := 0; col < pm.gridSize; col++ {
			partitionID := fmt.Sprintf("partition_%d_%d", row, col)
			bounds := protocol.Rectangle{
				X:      col * pm.partitionSize,
				Y:      row * pm.partitionSize,
				Width:  pm.partitionSize,
				Height: pm.partitionSize,
			}

			// Create actor props
			props := actor.PropsFromProducer(func() actor.Actor {
				return actor.NewPartitionActor(partitionID, bounds, pm.redisWriter)
			})

			// Spawn actor
			pid := pm.system.Root.Spawn(props)

			pm.partitions[partitionID] = pid
		}
	}
}

// GetPartitionPID returns the PID for a partition ID
func (pm *PartitionManager) GetPartitionPID(partitionID string) (*actor.PID, bool) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pid, exists := pm.partitions[partitionID]
	return pid, exists
}

// GetPartitionIDForPosition calculates partition ID for a position
func (pm *PartitionManager) GetPartitionIDForPosition(pos protocol.Point) string {
	row := pos.Y / pm.partitionSize
	col := pos.X / pm.partitionSize
	return fmt.Sprintf("partition_%d_%d", row, col)
}

// GetAllPartitionIDs returns all partition IDs
func (pm *PartitionManager) GetAllPartitionIDs() []string {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	ids := make([]string, 0, len(pm.partitions))
	for id := range pm.partitions {
		ids = append(ids, id)
	}
	return ids
}

// GetPartitionCount returns the number of partitions
func (pm *PartitionManager) GetPartitionCount() int {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return len(pm.partitions)
}

// StopAllPartitions stops all partition actors
func (pm *PartitionManager) StopAllPartitions() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	for id, pid := range pm.partitions {
		pm.system.Root.Stop(pid)
		delete(pm.partitions, id)
	}
}
