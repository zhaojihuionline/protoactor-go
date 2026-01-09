package main

import (
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"demo/internal/actor"
	"demo/internal/manager"
	"demo/internal/persistence"
	"demo/internal/protocol"
)

func TestIntegration(t *testing.T) {
	// Start actor system
	system := actor.NewActorSystem()

	// Start Redis writer
	redisWriterProps := actor.PropsFromProducer(func() actor.Actor {
		return persistence.NewRedisWriterActor("127.0.0.1:6379")
	})
	redisWriterPID := system.Root.Spawn(redisWriterProps)

	// Start partition manager and partitions
	pm := manager.NewPartitionManager(system, redisWriterPID)
	pm.StartAllPartitions()

	// Start event router
	eventRouterProps := actor.PropsFromProducer(func() actor.Actor {
		return actor.NewEventRouterActor(pm)
	})
	eventRouterPID := system.Root.Spawn(eventRouterProps)

	// Test player join
	playerID := "test_player"
	playerProps := actor.PropsFromProducer(func() actor.Actor {
		return actor.NewPlayerActor(playerID, "TestPlayer", protocol.Point{X: 50, Y: 50}, eventRouterPID)
	})
	playerPID := system.Root.Spawn(playerProps)

	// Create join event
	joinEvent := &protocol.PlayerJoinEvent{
		BaseEvent: protocol.BaseEvent{
			EventType: "player_join",
			Timestamp: time.Now(),
			SourceID:  playerID,
		},
		PlayerID:   playerID,
		PlayerName: "TestPlayer",
		Position:   protocol.Point{X: 50, Y: 50},
		OpID:       "test_join_op",
	}

	system.Root.Send(eventRouterPID, joinEvent)

	// Wait a bit for processing
	time.Sleep(time.Second)

	// Test move command via session
	sessionProps := actor.PropsFromProducer(func() actor.Actor {
		return actor.NewSessionActor(playerID, playerPID, eventRouterPID)
	})
	sessionPID := system.Root.Spawn(sessionProps)

	moveCmd := &protocol.MoveCommand{
		PlayerID: playerID,
		ToPos:    protocol.Point{X: 60, Y: 60},
		OpID:     "test_move_op",
	}

	system.Root.Send(sessionPID, moveCmd)

	// Wait for processing
	time.Sleep(time.Second * 2)

	// Check if persisted (in real test, verify Redis)
	log.Println("Integration test completed - check Redis for persisted data")

	// Cleanup
	pm.StopAllPartitions()
	system.Root.Stop(redisWriterPID)
	system.Root.Stop(eventRouterPID)
	system.Root.Stop(playerPID)
	system.Root.Stop(sessionPID)
}

func TestConcurrentPartitions(t *testing.T) {
	system := actor.NewActorSystem()

	redisWriterProps := actor.PropsFromProducer(func() actor.Actor {
		return persistence.NewRedisWriterActor("127.0.0.1:6379")
	})
	redisWriterPID := system.Root.Spawn(redisWriterProps)

	pm := manager.NewPartitionManager(system, redisWriterPID)
	pm.StartAllPartitions()

	eventRouterProps := actor.PropsFromProducer(func() actor.Actor {
		return actor.NewEventRouterActor(pm)
	})
	eventRouterPID := system.Root.Spawn(eventRouterProps)

	// Simulate concurrent operations on different partitions
	for i := 0; i < 10; i++ {
		playerID := fmt.Sprintf("player_%d", i)
		pos := protocol.Point{X: i * 100, Y: i * 100} // Different partitions

		playerProps := actor.PropsFromProducer(func() actor.Actor {
			return actor.NewPlayerActor(playerID, fmt.Sprintf("Player%d", i), pos, eventRouterPID)
		})
		playerPID := system.Root.Spawn(playerProps)

		joinEvent := &protocol.PlayerJoinEvent{
			BaseEvent: protocol.BaseEvent{
				EventType: "player_join",
				Timestamp: time.Now(),
				SourceID:  playerID,
			},
			PlayerID:   playerID,
			PlayerName: fmt.Sprintf("Player%d", i),
			Position:   pos,
			OpID:       fmt.Sprintf("join_op_%d", i),
		}

		system.Root.Send(eventRouterPID, joinEvent)

		// Move within partition
		go func(pid *actor.PID, id string, p protocol.Point) {
			time.Sleep(time.Millisecond * 100)
			newPos := protocol.Point{X: p.X + 10, Y: p.Y + 10}
			moveEvent := &protocol.EntityMoveEvent{
				BaseEvent: protocol.BaseEvent{
					EventType: "entity_move",
					Timestamp: time.Now(),
					SourceID:  id,
				},
				OpID:       fmt.Sprintf("move_op_%s", id),
				EntityID:   id,
				FromPos:    p,
				ToPos:      newPos,
				EntityType: protocol.EntityPlayer,
			}
			system.Root.Send(eventRouterPID, moveEvent)
		}(playerPID, playerID, pos)
	}

	// Wait for all operations
	time.Sleep(time.Second * 3)

	log.Println("Concurrent test completed")

	// Cleanup
	pm.StopAllPartitions()
	system.Root.Stop(redisWriterPID)
	system.Root.Stop(eventRouterPID)
}

func BenchmarkPartitionOperations(b *testing.B) {
	system := actor.NewActorSystem()

	redisWriterProps := actor.PropsFromProducer(func() actor.Actor {
		return persistence.NewRedisWriterActor("127.0.0.1:6379")
	})
	redisWriterPID := system.Root.Spawn(redisWriterProps)

	pm := manager.NewPartitionManager(system, redisWriterPID)
	pm.StartAllPartitions()

	eventRouterProps := actor.PropsFromProducer(func() actor.Actor {
		return actor.NewEventRouterActor(pm)
	})
	eventRouterPID := system.Root.Spawn(eventRouterProps)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		playerID := fmt.Sprintf("bench_player_%d", i)
		pos := protocol.Point{X: i % 1200, Y: i % 1200}

		joinEvent := &protocol.PlayerJoinEvent{
			BaseEvent: protocol.BaseEvent{
				EventType: "player_join",
				Timestamp: time.Now(),
				SourceID:  playerID,
			},
			PlayerID:   playerID,
			PlayerName: fmt.Sprintf("BenchPlayer%d", i),
			Position:   pos,
			OpID:       fmt.Sprintf("bench_join_%d", i),
		}

		system.Root.Send(eventRouterPID, joinEvent)
	}

	b.StopTimer()

	// Cleanup
	pm.StopAllPartitions()
	system.Root.Stop(redisWriterPID)
	system.Root.Stop(eventRouterPID)
}
