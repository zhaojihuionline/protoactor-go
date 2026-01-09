package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"demo/internal/actor"
	"demo/internal/protocol"
)

func main() {
	// Connect to server (simplified - in real implementation, use network)
	system := actor.NewActorSystem()

	// Simulate connecting to server
	playerID := "player_1"
	playerName := "TestPlayer"

	// Create player actor
	eventRouterPID := actor.NewPID("127.0.0.1:0", "event_router") // Placeholder
	playerProps := actor.PropsFromProducer(func() actor.Actor {
		return actor.NewPlayerActor(playerID, playerName, protocol.Point{X: 50, Y: 50}, eventRouterPID)
	})
	playerPID := system.Root.Spawn(playerProps)

	// Create session actor
	sessionProps := actor.PropsFromProducer(func() actor.Actor {
		return actor.NewSessionActor(playerID, playerPID, eventRouterPID)
	})
	sessionPID := system.Root.Spawn(sessionProps)

	// Start AOI listener
	go listenForAOIUpdates(sessionPID)

	// CLI loop
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Demo CLI Client")
	fmt.Println("Commands: move <x> <y>, interact <target>, quit")

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "quit" {
			break
		}

		parts := strings.Split(line, " ")
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "move":
			if len(parts) == 3 {
				x, errX := strconv.Atoi(parts[1])
				y, errY := strconv.Atoi(parts[2])
				if errX == nil && errY == nil {
					cmd := &protocol.MoveCommand{
						PlayerID: playerID,
						ToPos:    protocol.Point{X: x, Y: y},
						OpID:     fmt.Sprintf("op_%d", time.Now().UnixNano()),
					}
					system.Root.Send(sessionPID, cmd)
					fmt.Printf("Sent move command to (%d,%d)\n", x, y)
				} else {
					fmt.Println("Invalid coordinates")
				}
			} else {
				fmt.Println("Usage: move <x> <y>")
			}
		case "interact":
			if len(parts) == 2 {
				targetID := parts[1]
				cmd := &protocol.InteractCommand{
					PlayerID: playerID,
					TargetID: targetID,
					Action:   "talk", // Default action
					OpID:     fmt.Sprintf("op_%d", time.Now().UnixNano()),
				}
				system.Root.Send(sessionPID, cmd)
				fmt.Printf("Sent interact command to %s\n", targetID)
			} else {
				fmt.Println("Usage: interact <target_id>")
			}
		default:
			fmt.Println("Unknown command. Use: move <x> <y>, interact <target>, quit")
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}

	// Cleanup
	system.Root.Stop(playerPID)
	system.Root.Stop(sessionPID)
}

// listenForAOIUpdates listens for AOI updates from session actor
func listenForAOIUpdates(sessionPID *actor.PID) {
	// In a real implementation, this would be part of the session actor's outbound channel
	// For demo, simulate receiving updates
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for range ticker.C {
		// Simulate receiving AOI update
		update := &protocol.AOIUpdate{
			PlayerID: "player_1",
			VisibleEntities: []protocol.VisibleEntity{
				{ID: "npc_1", Position: protocol.Point{X: 60, Y: 60}},
			},
			VisibleGrids: []protocol.GridState{},
		}
		fmt.Printf("Received AOI update: %d visible entities\n", len(update.VisibleEntities))
	}
}
