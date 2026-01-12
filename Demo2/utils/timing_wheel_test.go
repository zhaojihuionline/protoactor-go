package utils

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTimingWheel_Basic(t *testing.T) {
	tw := NewTimingWheel()
	err := tw.Start()
	if err != nil {
		t.Fatalf("Failed to start timing wheel: %v", err)
	}
	defer tw.Stop()

	// 测试毫秒级任务
	executed := make(chan bool, 1)
	taskID, err := tw.AddTimer(200*time.Millisecond, func() {
		executed <- true
	})

	if err != nil {
		t.Fatalf("Failed to add timer: %v", err)
	}

	if taskID == "" {
		t.Fatal("Task ID should not be empty")
	}

	// 等待任务执行
	select {
	case <-executed:
		// 成功
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timer did not execute within expected time")
	}
}

func TestTimingWheel_Cancel(t *testing.T) {
	tw := NewTimingWheel()
	err := tw.Start()
	if err != nil {
		t.Fatalf("Failed to start timing wheel: %v", err)
	}
	defer tw.Stop()

	executed := int32(0)

	// 添加任务
	taskID, err := tw.AddTimer(100*time.Millisecond, func() {
		atomic.AddInt32(&executed, 1)
	})

	if err != nil {
		t.Fatalf("Failed to add timer: %v", err)
	}

	// 立即取消
	cancelled := tw.CancelTimer(taskID)
	if !cancelled {
		t.Fatal("Failed to cancel timer")
	}

	// 等待一段时间，确保任务没有执行
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&executed) != 0 {
		t.Fatal("Cancelled timer should not have executed")
	}
}

func TestTimingWheel_DifferentLevels(t *testing.T) {
	tw := NewTimingWheel()
	err := tw.Start()
	if err != nil {
		t.Fatalf("Failed to start timing wheel: %v", err)
	}
	defer tw.Stop()

	testCases := []struct {
		name     string
		delay    time.Duration
		expected bool
	}{
		{"millisecond", 100 * time.Millisecond, true},
		{"second", 500 * time.Millisecond, true},
		{"minute", 2 * time.Second, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executed := make(chan bool, 1)

			_, err := tw.AddTimer(tc.delay, func() {
				executed <- true
			})

			if err != nil {
				t.Fatalf("Failed to add %s timer: %v", tc.name, err)
			}

			// 等待任务执行，多给一些宽限时间
			timeout := tc.delay + 200*time.Millisecond
			select {
			case <-executed:
				// 成功
			case <-time.After(timeout):
				t.Fatalf("%s timer did not execute within %v", tc.name, timeout)
			}
		})
	}
}

func TestTimingWheel_Concurrent(t *testing.T) {
	tw := NewTimingWheel()
	err := tw.Start()
	if err != nil {
		t.Fatalf("Failed to start timing wheel: %v", err)
	}
	defer tw.Stop()

	const numGoroutines = 100
	const tasksPerGoroutine = 10

	var executed int64
	var wg sync.WaitGroup

	// 启动多个goroutine并发添加任务
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < tasksPerGoroutine; j++ {
				delay := time.Duration(50+rand.Intn(100)) * time.Millisecond
				_, err := tw.AddTimer(delay, func() {
					atomic.AddInt64(&executed, 1)
				})

				if err != nil {
					t.Errorf("Failed to add timer: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// 等待所有任务执行 (增加等待时间，因为有些任务可能在秒层)
	time.Sleep(2 * time.Second)

	finalExecuted := atomic.LoadInt64(&executed)
	expected := int64(numGoroutines * tasksPerGoroutine)

	if finalExecuted != expected {
		t.Fatalf("Expected %d tasks to execute, but got %d", expected, finalExecuted)
	}
}

func TestTimingWheel_Stats(t *testing.T) {
	tw := NewTimingWheel()
	err := tw.Start()
	if err != nil {
		t.Fatalf("Failed to start timing wheel: %v", err)
	}
	defer tw.Stop()

	// 添加一些任务
	for i := 0; i < 10; i++ {
		_, _ = tw.AddTimer(100*time.Millisecond, func() {})
	}

	time.Sleep(50 * time.Millisecond) // 让任务有机会被处理

	stats := tw.GetStats()

	if stats["total_tasks"].(int64) < 0 {
		t.Error("Total tasks should be non-negative")
	}

	if !stats["running"].(bool) {
		t.Error("Timing wheel should be running")
	}

	// 检查是否有层级统计
	levelNames := []string{"week", "day", "hour", "minute", "second", "millisecond"}
	for _, levelName := range levelNames {
		tasksKey := levelName + "_tasks"
		if _, exists := stats[tasksKey]; !exists {
			t.Errorf("Missing stats key: %s", tasksKey)
		}
	}
}

func TestTimingWheel_Stop(t *testing.T) {
	tw := NewTimingWheel()

	err := tw.Start()
	if err != nil {
		t.Fatalf("Failed to start timing wheel: %v", err)
	}

	// 添加一个长时间任务
	_, err = tw.AddTimer(1*time.Hour, func() {
		t.Log("This should not execute after stop")
	})

	if err != nil {
		t.Fatalf("Failed to add timer: %v", err)
	}

	// 停止
	err = tw.Stop()
	if err != nil {
		t.Fatalf("Failed to stop timing wheel: %v", err)
	}

	// 再次停止应该失败
	err = tw.Stop()
	if err == nil {
		t.Error("Stopping an already stopped timing wheel should fail")
	}
}

func BenchmarkTimingWheel_AddTimer(b *testing.B) {
	tw := NewTimingWheel()
	tw.Start()
	defer tw.Stop()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tw.AddTimer(100*time.Millisecond, func() {})
		}
	})
}

func BenchmarkTimingWheel_AddAndCancel(b *testing.B) {
	tw := NewTimingWheel()
	tw.Start()
	defer tw.Stop()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			taskID, _ := tw.AddTimer(100*time.Millisecond, func() {})
			tw.CancelTimer(taskID)
		}
	})
}
