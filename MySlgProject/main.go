package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 加载配置
	config := LoadConfig()

	fmt.Printf("Starting SLG Game Server...\n")
	fmt.Printf("World Size: %dx%d\n", config.World.Width, config.World.Height)
	fmt.Printf("AOI Range: %d\n", config.AOI.ViewRange)
	fmt.Printf("Partition Size: %d\n", config.Partition.DefaultSize)

	// 创建游戏世界
	gameWorld := NewGameWorld(config)

	// 启动游戏世界
	if err := gameWorld.Start(); err != nil {
		log.Fatalf("Failed to start game world: %v", err)
	}

	// 启动HTTP监控服务器
	go startMonitoringServer(gameWorld)

	// 添加一些测试玩家
	addTestPlayers(gameWorld)

	// 设置信号处理
	setupSignalHandler(gameWorld)

	// 主循环
	fmt.Println("SLG Game Server is running. Press Ctrl+C to stop.")

	// 阻塞等待信号
	select {}
}

// startMonitoringServer 启动监控服务器
func startMonitoringServer(gameWorld *GameWorld) {
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		stats := gameWorld.GetSystemStats()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// 简单的JSON输出（实际应该使用json.Marshal）
		fmt.Fprintf(w, `{
  "world": {
    "total_players": %d,
    "total_entities": %d,
    "total_partitions": %d,
    "uptime_seconds": %.0f
  },
  "entities": {
    "total": %d
  },
  "partitions": {
    "active": %d
  }
}`,
			stats["world"].(WorldStats).TotalPlayers,
			stats["world"].(WorldStats).TotalEntities,
			stats["world"].(WorldStats).TotalPartitions,
			stats["world"].(WorldStats).Uptime.Seconds(),
			stats["entities"].(map[string]interface{})["total_entities"],
			stats["partitions"].(map[string]interface{})["count"],
		)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})

	fmt.Println("Monitoring server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Printf("Monitoring server failed: %v", err)
	}
}

// addTestPlayers 添加测试玩家
func addTestPlayers(gameWorld *GameWorld) {
	testPlayers := []struct {
		id   string
		name string
		pos  Point
	}{
		{"player_001", "Alice", Point{X: 100, Y: 100}},
		{"player_002", "Bob", Point{X: 200, Y: 200}},
		{"player_003", "Charlie", Point{X: 300, Y: 300}},
		{"player_004", "Diana", Point{X: 400, Y: 400}},
		{"player_005", "Eve", Point{X: 500, Y: 500}},
	}

	for _, player := range testPlayers {
		if _, err := gameWorld.AddPlayer(player.id, player.name, player.pos); err != nil {
			log.Printf("Failed to add test player %s: %v", player.id, err)
		} else {
			log.Printf("Added test player: %s at (%d,%d)", player.name, player.pos.X, player.pos.Y)
		}
	}

	// 模拟玩家移动
	go simulatePlayerMovement(gameWorld, testPlayers)
}

// simulatePlayerMovement 模拟玩家移动
func simulatePlayerMovement(gameWorld *GameWorld, players []struct {
	id   string
	name string
	pos  Point
}) {
	ticker := time.NewTicker(time.Second * 3) // 每3秒移动一次
	defer ticker.Stop()

	directions := []Point{
		{X: 1, Y: 0},   // 右
		{X: -1, Y: 0},  // 左
		{X: 0, Y: 1},   // 上
		{X: 0, Y: -1},  // 下
		{X: 1, Y: 1},   // 右上
		{X: 1, Y: -1},  // 右下
		{X: -1, Y: 1},  // 左上
		{X: -1, Y: -1}, // 左下
	}

	for range ticker.C {
		for i := range players {
			player := &players[i]

			// 随机选择移动方向
			dir := directions[time.Now().UnixNano()%int64(len(directions))]

			// 计算新位置
			newX := player.pos.X + dir.X*10 // 每次移动10格
			newY := player.pos.Y + dir.Y*10

			// 边界检查
			if newX < 0 {
				newX = 0
			}
			if newX >= gameWorld.config.World.Width {
				newX = gameWorld.config.World.Width - 1
			}
			if newY < 0 {
				newY = 0
			}
			if newY >= gameWorld.config.World.Height {
				newY = gameWorld.config.World.Height - 1
			}

			newPos := Point{X: newX, Y: newY}

			// 更新位置
			if err := gameWorld.UpdatePlayerPosition(player.id, newPos); err != nil {
				log.Printf("Failed to move player %s: %v", player.id, err)
			} else {
				player.pos = newPos
				log.Printf("Player %s moved to (%d,%d)", player.name, newPos.X, newPos.Y)
			}
		}
	}
}

// setupSignalHandler 设置信号处理
func setupSignalHandler(gameWorld *GameWorld) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nReceived shutdown signal. Shutting down gracefully...")

		// 执行清理操作
		gameWorld.Shutdown()

		fmt.Println("Shutdown complete.")
		os.Exit(0)
	}()
}

// performanceTest 性能测试
func performanceTest(gameWorld *GameWorld) {
	fmt.Println("Starting performance test...")

	// 添加大量测试玩家
	start := time.Now()
	playerCount := 1000

	for i := 0; i < playerCount; i++ {
		playerID := fmt.Sprintf("perf_player_%03d", i)
		playerName := fmt.Sprintf("PerfPlayer%d", i)
		pos := Point{
			X: i % gameWorld.config.World.Width,
			Y: (i / gameWorld.config.World.Width) % gameWorld.config.World.Height,
		}

		if _, err := gameWorld.AddPlayer(playerID, playerName, pos); err != nil {
			log.Printf("Failed to add performance test player %s: %v", playerID, err)
		}
	}

	elapsed := time.Since(start)
	fmt.Printf("Added %d players in %v (%.0f players/sec)\n",
		playerCount, elapsed, float64(playerCount)/elapsed.Seconds())

	// 测试AOI查询性能
	aoiStart := time.Now()
	for i := 0; i < 100; i++ {
		testPos := Point{X: i * 10, Y: i * 10}
		entities := gameWorld.entityMgr.GetEntitiesInRange(testPos, gameWorld.config.AOI.ViewRange)
		_ = entities // 避免未使用变量警告
	}
	aoiElapsed := time.Since(aoiStart)
	fmt.Printf("AOI queries completed in %v (%.0f queries/sec)\n",
		aoiElapsed, 100.0/aoiElapsed.Seconds())

	// 显示最终统计信息
	stats := gameWorld.GetSystemStats()
	fmt.Printf("Final stats: %d players, %d entities, %d partitions\n",
		stats["world"].(WorldStats).TotalPlayers,
		stats["world"].(WorldStats).TotalEntities,
		stats["world"].(WorldStats).TotalPartitions)
}
