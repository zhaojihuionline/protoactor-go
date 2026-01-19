# Universal Timing Wheel (分层时间轮)

通用分层时间轮实现，支持毫秒到周的时间范围，专为游戏服务器设计。

## 特性

- **6层分层设计**: 毫秒 → 秒 → 分钟 → 小时 → 天 → 周
- **高并发安全**: 使用细粒度锁保证并发访问安全
- **自动层级选择**: 智能选择最适合的时间层级
- **任务降级**: 支持任务在层级间自动降级
- **内存高效**: 稀疏存储，只为有任务的槽位分配内存
- **取消支持**: 支持任务取消操作
- **统计监控**: 提供详细的运行时统计信息

## 架构设计

### 时间层级

| 层级 | 时间单位 | 槽位数 | 覆盖范围 | 适用场景 |
|------|----------|--------|----------|----------|
| 毫秒层 | 100ms | 10 | 1秒 | 技能特效、动画 |
| 秒层 | 1秒 | 60 | 1分钟 | AI决策、状态更新 |
| 分钟层 | 1分钟 | 60 | 1小时 | 建筑建造、战斗冷却 |
| 小时层 | 1小时 | 24 | 1天 | 玩家恢复、事件触发 |
| 天层 | 1天 | 7 | 1周 | 日常任务、服务器维护 |
| 周层 | 1周 | 12 | 3个月 | 长期活动、版本更新 |

### 并发安全设计

- **全局锁**: 保护 `timers` map
- **槽位锁**: 每个槽位独立锁，支持并发访问
- **原子操作**: 使用原子操作更新计数器

## 使用方法

### 基本使用

```go
package main

import (
    "fmt"
    "time"
    "demo/utils"
)

func main() {
    // 创建时间轮
    tw := utils.NewTimingWheel()

    // 启动
    err := tw.Start()
    if err != nil {
        panic(err)
    }
    defer tw.Stop()

    // 添加定时任务
    taskID, err := tw.AddTimer(5*time.Second, func() {
        fmt.Println("5秒后执行的任务")
    })

    if err != nil {
        fmt.Printf("添加任务失败: %v\n", err)
        return
    }

    fmt.Printf("任务已添加，ID: %s\n", taskID)

    // 取消任务
    // tw.CancelTimer(taskID)

    // 等待执行
    time.Sleep(6 * time.Second)
}
```

### 游戏场景示例

```go
// 技能冷却
func scheduleSkillCooldown(playerID string, skillID string, cooldown time.Duration) {
    tw.AddTimer(cooldown, func() {
        // 重置技能冷却
        resetPlayerSkillCooldown(playerID, skillID)
    })
}

// 建筑建造
func scheduleBuildingConstruction(buildingID string, buildTime time.Duration) {
    tw.AddTimer(buildTime, func() {
        // 完成建筑建造
        completeBuilding(buildingID)
    })
}

// NPC AI决策
func scheduleNPCAI(npcID string) {
    interval := 3 * time.Second // 3秒决策一次
    tw.AddTimer(interval, func() {
        // 执行AI逻辑
        npc.MakeDecision()
        // 递归安排下一次
        scheduleNPCAI(npcID)
    })
}

// 每日重置
func scheduleDailyReset() {
    // 计算到明天0点的延迟
    now := time.Now()
    tomorrow := now.AddDate(0, 0, 1)
    resetTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
                          0, 0, 0, 0, tomorrow.Location())
    delay := resetTime.Sub(now)

    tw.AddTimer(delay, func() {
        // 执行每日重置逻辑
        performDailyReset()
        // 递归安排明天
        scheduleDailyReset()
    })
}
```

### 监控统计

```go
// 获取运行统计
stats := tw.GetStats()
fmt.Printf("总任务数: %d\n", stats["total_tasks"])
fmt.Printf("运行时间: %v\n", stats["uptime"])
fmt.Printf("毫秒层任务: %d\n", stats["millisecond_tasks"])
fmt.Printf("秒层任务: %d\n", stats["second_tasks"])
// ... 其他层级
```

## 性能特性

### 时间复杂度
- **添加任务**: O(1) - 直接计算层级和槽位
- **取消任务**: O(N) - 需要遍历槽位找到任务，最坏情况
- **执行到期任务**: O(K) - K为到期任务数

### 内存使用
- **基础开销**: ~6KB (6个层级的基础结构)
- **每任务开销**: ~100字节 (TimerTask + map条目)
- **稀疏存储**: 只为有任务的槽位分配内存

### 并发性能
- **读操作**: 高效，支持高并发读取
- **写操作**: 通过细粒度锁保证安全
- **扩容友好**: 支持动态调整参数

## 配置调优

### 层级配置

```go
// 可以根据需要调整层级参数
customLevels := []*TimeWheelLevel{
    // 自定义毫秒层：50ms精度，20槽位
    {name: "millisecond", unit: 50*time.Millisecond, slotsPerUnit: 20, totalSlots: 20},
    // 其他层级...
}
```

### 内存优化

```go
// 对于内存敏感场景，可以减少槽位数
memoryOptimizedLevels := []*TimeWheelLevel{
    {name: "second", unit: time.Second, slotsPerUnit: 1, totalSlots: 30}, // 30秒
    {name: "minute", unit: time.Minute, slotsPerUnit: 1, totalSlots: 30}, // 30分钟
    // ... 其他层级
}
```

## 测试覆盖

### 单元测试
- ✅ 基本定时功能
- ✅ 任务取消
- ✅ 不同时间层级
- ✅ 高并发访问
- ✅ 统计信息

### 基准测试
- 添加任务性能
- 并发读写性能
- 内存使用情况

## 使用建议

### 游戏服务器场景
1. **技能冷却**: 使用秒/分钟层
2. **建筑建造**: 使用分钟/小时层
3. **每日重置**: 使用小时/天层
4. **长期活动**: 使用天/周层

### 最佳实践
1. **合理分层**: 根据业务需求选择合适的层级
2. **监控统计**: 定期检查各层级的任务分布
3. **错误处理**: 添加任务失败时的降级处理
4. **资源清理**: 及时取消不需要的任务

### 注意事项
1. **时间精度**: 毫秒层提供最高精度，但消耗更多CPU
2. **内存监控**: 高负载时注意各层级的任务分布
3. **并发限制**: 避免单点过热，合理分布任务

这个分层时间轮实现提供了从毫秒到月的完整时间范围，特别适合游戏服务器这种需要处理各种时间间隔任务的场景。
