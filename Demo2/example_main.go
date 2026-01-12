package main

import (
	"fmt"
	"log"
	"time"

	"demo/utils"
)

func main() {
	fmt.Println("=== 通用分层时间轮演示 ===")

	// 创建时间轮
	tw := utils.NewTimingWheel()

	// 启动时间轮
	err := tw.Start()
	if err != nil {
		log.Fatalf("启动时间轮失败: %v", err)
	}
	defer tw.Stop()

	fmt.Println("时间轮已启动，开始添加演示任务...")

	// 1. 毫秒级任务 - 技能特效
	fmt.Println("\n1. 添加毫秒级任务 (技能特效)")
	taskID1, err := tw.AddTimer(500*time.Millisecond, func() {
		fmt.Printf("[%s] ⚡ 技能特效完成！\n", time.Now().Format("15:04:05.000"))
	})
	if err != nil {
		log.Printf("添加技能特效任务失败: %v", err)
	} else {
		fmt.Printf("技能特效任务ID: %s\n", taskID1)
	}

	// 2. 秒级任务 - AI决策
	fmt.Println("\n2. 添加秒级任务 (AI决策)")
	taskID2, err := tw.AddTimer(2*time.Second, func() {
		fmt.Printf("[%s] 🤖 怪物AI决策完成！\n", time.Now().Format("15:04:05.000"))
	})
	if err != nil {
		log.Printf("添加AI决策任务失败: %v", err)
	} else {
		fmt.Printf("AI决策任务ID: %s\n", taskID2)
	}

	// 3. 分钟级任务 - 建筑建造
	fmt.Println("\n3. 添加分钟级任务 (建筑建造)")
	taskID3, err := tw.AddTimer(5*time.Second, func() { // 为演示改成5秒，实际应该是分钟
		fmt.Printf("[%s] 🏗️ 建筑建造完成！\n", time.Now().Format("15:04:05.000"))
	})
	if err != nil {
		log.Printf("添加建筑建造任务失败: %v", err)
	} else {
		fmt.Printf("建筑建造任务ID: %s\n", taskID3)
	}

	// 4. 演示任务取消
	fmt.Println("\n4. 演示任务取消")
	taskID4, err := tw.AddTimer(3*time.Second, func() {
		fmt.Printf("[%s] ❌ 这个任务应该被取消\n", time.Now().Format("15:04:05.000"))
	})
	if err != nil {
		log.Printf("添加待取消任务失败: %v", err)
	} else {
		fmt.Printf("待取消任务ID: %s\n", taskID4)

		// 1秒后取消任务
		time.Sleep(1 * time.Second)
		cancelled := tw.CancelTimer(taskID4)
		if cancelled {
			fmt.Printf("✅ 任务 %s 已成功取消\n", taskID4)
		} else {
			fmt.Printf("❌ 任务 %s 取消失败\n", taskID4)
		}
	}

	// 等待任务执行
	fmt.Println("\n5. 等待任务执行...")
	time.Sleep(6 * time.Second)

	// 显示统计信息
	fmt.Println("\n6. 时间轮统计信息:")
	stats := tw.GetStats()
	fmt.Printf("总任务数: %d\n", stats["total_tasks"])
	fmt.Printf("运行状态: %v\n", stats["running"])
	fmt.Printf("运行时间: %v\n", stats["uptime"])

	// 显示各层级统计
	levelNames := []string{"millisecond", "second", "minute", "hour", "day", "week"}
	for _, level := range levelNames {
		tasks := stats[level+"_tasks"]
		ticks := stats[level+"_tick_count"]
		fmt.Printf("%s层 - 任务数: %d, Tick数: %d\n", level, tasks, ticks)
	}

	fmt.Println("\n=== 演示完成 ===")
}
