package main

import (
	"sort"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

// LoadBalancer 负载均衡器 - 监控分区负载并触发调整
type LoadBalancer struct {
	partitions     map[string]*PartitionMetrics
	thresholds     LoadThresholds
	adjustTimer    *time.Timer
	mutex          sync.RWMutex
	config         *Config
	system         *actor.ActorSystem
	partitionMgr   *PartitionManager
	adjuster       *PartitionAdjuster
	lastAdjustment time.Time
	history        []AdjustmentRecord
}

// AdjustmentRecord 调整记录
type AdjustmentRecord struct {
	Timestamp    time.Time
	Action       AdjustmentAction
	PartitionID  string
	Reason       string
	BeforeState  *PartitionMetrics
	AfterState   *PartitionMetrics
	Success      bool
}

// NewLoadBalancer 创建负载均衡器
func NewLoadBalancer(config *Config, system *actor.ActorSystem,
	partitionMgr *PartitionManager, adjuster *PartitionAdjuster) *LoadBalancer {

	lb := &LoadBalancer{
		partitions:     make(map[string]*PartitionMetrics),
		thresholds:     config.Partition.LoadThresholds,
		config:         config,
		system:         system,
		partitionMgr:   partitionMgr,
		adjuster:       adjuster,
		lastAdjustment: time.Now(),
		history:        make([]AdjustmentRecord, 0),
	}

	// 启动监控定时器
	lb.adjustTimer = time.AfterFunc(config.Partition.MonitorInterval, lb.monitorAndAdjust)

	return lb
}

// Start 启动负载均衡器
func (lb *LoadBalancer) Start() {
	// 初始化分区指标
	lb.collectInitialMetrics()
}

// Stop 停止负载均衡器
func (lb *LoadBalancer) Stop() {
	if lb.adjustTimer != nil {
		lb.adjustTimer.Stop()
	}
}

// UpdateMetrics 更新分区指标
func (lb *LoadBalancer) UpdateMetrics(partitionID string, metrics *PartitionMetrics) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.partitions[partitionID] = metrics
}

// GetMetrics 获取分区指标
func (lb *LoadBalancer) GetMetrics(partitionID string) (*PartitionMetrics, bool) {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	metrics, exists := lb.partitions[partitionID]
	return metrics, exists
}

// GetAllMetrics 获取所有分区指标
func (lb *LoadBalancer) GetAllMetrics() map[string]*PartitionMetrics {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	result := make(map[string]*PartitionMetrics)
	for id, metrics := range lb.partitions {
		result[id] = metrics
	}
	return result
}

// collectInitialMetrics 收集初始指标
func (lb *LoadBalancer) collectInitialMetrics() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	for partitionID, partition := range lb.partitionMgr.GetAllPartitions() {
		metrics := partition.GetMetrics()
		lb.partitions[partitionID] = metrics
	}
}

// monitorAndAdjust 监控并调整
func (lb *LoadBalancer) monitorAndAdjust() {
	defer func() {
		// 重新调度下一次检查
		lb.adjustTimer.Reset(lb.config.Partition.MonitorInterval)
	}()

	// 收集最新指标
	lb.collectMetrics()

	// 检查是否需要调整
	if lb.needsAdjustment() {
		plans := lb.calculateAdjustments()

		// 执行调整计划
		for _, plan := range plans {
			lb.executeAdjustment(plan)
		}
	}
}

// collectMetrics 收集指标
func (lb *LoadBalancer) collectMetrics() {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	for partitionID, partition := range lb.partitionMgr.GetAllPartitions() {
		metrics := partition.GetMetrics()
		lb.partitions[partitionID] = metrics
	}
}

// needsAdjustment 检查是否需要调整
func (lb *LoadBalancer) needsAdjustment() bool {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	// 检查是否在调整冷却期内
	if time.Since(lb.lastAdjustment) < lb.thresholds.AdjustmentCooldown {
		return false
	}

	for _, metrics := range lb.partitions {
		// 检查过载条件
		if lb.isOverloaded(metrics) {
			return true
		}

		// 检查低效条件
		if lb.isUnderloaded(metrics) {
			return true
		}
	}

	return false
}

// isOverloaded 检查是否过载
func (lb *LoadBalancer) isOverloaded(metrics *PartitionMetrics) bool {
	return metrics.PlayerCount > lb.thresholds.MaxPlayersPerPartition ||
		   (metrics.MessageRate > 0 && metrics.MessageRate > lb.thresholds.MaxMessageRate) ||
		   (metrics.CpuUsage > 0 && metrics.CpuUsage > lb.thresholds.MaxCpuUsage) ||
		   (metrics.MemoryUsage > 0 && metrics.MemoryUsage > lb.thresholds.MaxMemoryUsage)
}

// isUnderloaded 检查是否低效
func (lb *LoadBalancer) isUnderloaded(metrics *PartitionMetrics) bool {
	// 玩家数量过少且持续时间较长
	if metrics.PlayerCount < lb.thresholds.MinPlayersPerPartition {
		// 检查是否持续低负载（5分钟）
		if time.Since(metrics.LastUpdate) > 5*time.Minute {
			return true
		}
	}

	// 资源利用率过低
	if metrics.CpuUsage > 0 && metrics.CpuUsage < lb.thresholds.MinCpuUsage &&
	   metrics.MemoryUsage > 0 && metrics.MemoryUsage < lb.thresholds.MinCpuUsage {
		return true
	}

	return false
}

// calculateAdjustments 计算调整计划
func (lb *LoadBalancer) calculateAdjustments() []*AdjustmentPlan {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	plans := make([]*AdjustmentPlan, 0)

	// 识别过载分区
	overloaded := lb.findOverloadedPartitions()

	// 识别低效分区
	underloaded := lb.findUnderloadedPartitions()

	// 生成分割计划
	for _, partitionID := range overloaded {
		if plan := lb.createSplitPlan(partitionID); plan != nil {
			plans = append(plans, plan)
		}
	}

	// 生成合并计划
	if len(underloaded) >= 2 {
		mergePlans := lb.createMergePlans(underloaded)
		plans = append(plans, mergePlans...)
	}

	// 优化调整计划（避免冲突，排序优先级）
	lb.optimizePlans(plans)

	return plans
}

// findOverloadedPartitions 查找过载分区
func (lb *LoadBalancer) findOverloadedPartitions() []string {
	overloaded := make([]string, 0)

	for partitionID, metrics := range lb.partitions {
		if lb.isOverloaded(metrics) {
			overloaded = append(overloaded, partitionID)
		}
	}

	return overloaded
}

// findUnderloadedPartitions 查找低效分区
func (lb *LoadBalancer) findUnderloadedPartitions() []string {
	underloaded := make([]string, 0)

	for partitionID, metrics := range lb.partitions {
		if lb.isUnderloaded(metrics) {
			underloaded = append(underloaded, partitionID)
		}
	}

	return underloaded
}

// createSplitPlan 创建分割计划
func (lb *LoadBalancer) createSplitPlan(partitionID string) *AdjustmentPlan {
	partition, exists := lb.partitionMgr.GetPartition(partitionID)
	if !exists {
		return nil
	}

	// 检查是否可以分割
	if partition.Size <= lb.config.Partition.MinSize*2 {
		return nil // 太小无法分割
	}

	return &AdjustmentPlan{
		PartitionID: partitionID,
		Action:      ActionSplit,
		NewSize:     partition.Size / 2,
		Priority:    lb.calculateSplitPriority(partitionID),
		EstimatedCost: 2 * time.Second, // 估算的调整时间
	}
}

// createMergePlans 创建合并计划
func (lb *LoadBalancer) createMergePlans(underloaded []string) []*AdjustmentPlan {
	plans := make([]*AdjustmentPlan, 0)

	// 按地理位置分组相邻的分区
	groups := lb.groupAdjacentPartitions(underloaded)

	for _, group := range groups {
		if len(group) >= 2 {
			plan := &AdjustmentPlan{
				PartitionID:        group[0], // 主分区
				Action:             ActionMerge,
				AffectedPartitions: group[1:],
				Priority:           lb.calculateMergePriority(group),
				EstimatedCost:      time.Duration(len(group)) * time.Second,
			}
			plans = append(plans, plan)
		}
	}

	return plans
}

// groupAdjacentPartitions 分组相邻分区
func (lb *LoadBalancer) groupAdjacentPartitions(partitionIDs []string) [][]string {
	// 简化的实现：将所有低效分区分为一组
	// 实际实现应该基于地理邻接性
	if len(partitionIDs) <= 4 {
		return [][]string{partitionIDs}
	}

	// 分成多个小组，每组最多4个
	groups := make([][]string, 0)
	for i := 0; i < len(partitionIDs); i += 4 {
		end := i + 4
		if end > len(partitionIDs) {
			end = len(partitionIDs)
		}
		groups = append(groups, partitionIDs[i:end])
	}

	return groups
}

// calculateSplitPriority 计算分割优先级
func (lb *LoadBalancer) calculateSplitPriority(partitionID string) int {
	metrics := lb.partitions[partitionID]
	if metrics == nil {
		return 0
	}

	// 基于过载程度计算优先级
	priority := 0

	if metrics.PlayerCount > lb.thresholds.MaxPlayersPerPartition*2 {
		priority += 10
	} else if metrics.PlayerCount > lb.thresholds.MaxPlayersPerPartition {
		priority += 5
	}

	if metrics.MessageRate > lb.thresholds.MaxMessageRate*2 {
		priority += 10
	} else if metrics.MessageRate > lb.thresholds.MaxMessageRate {
		priority += 5
	}

	return priority
}

// calculateMergePriority 计算合并优先级
func (lb *LoadBalancer) calculateMergePriority(group []string) int {
	totalPlayers := 0
	for _, partitionID := range group {
		if metrics := lb.partitions[partitionID]; metrics != nil {
			totalPlayers += metrics.PlayerCount
		}
	}

	// 如果总玩家数仍然很低，优先级更高
	if totalPlayers < lb.thresholds.MinPlayersPerPartition {
		return 8
	}

	return 3
}

// optimizePlans 优化调整计划
func (lb *LoadBalancer) optimizePlans(plans []*AdjustmentPlan) {
	// 按优先级排序
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].Priority > plans[j].Priority
	})

	// 限制同时调整的数量（避免系统过载）
	maxConcurrent := 3
	if len(plans) > maxConcurrent {
		plans = plans[:maxConcurrent]
	}

	// 检查计划冲突并解决
	lb.resolveConflicts(plans)
}

// resolveConflicts 解决计划冲突
func (lb *LoadBalancer) resolveConflicts(plans []*AdjustmentPlan) {
	// 简化的实现：移除涉及相同分区的冲突计划
	usedPartitions := make(map[string]bool)
	validPlans := make([]*AdjustmentPlan, 0)

	for _, plan := range plans {
		conflicts := false

		// 检查主分区
		if usedPartitions[plan.PartitionID] {
			conflicts = true
		}

		// 检查受影响的分区
		for _, affectedID := range plan.AffectedPartitions {
			if usedPartitions[affectedID] {
				conflicts = true
				break
			}
		}

		if !conflicts {
			validPlans = append(validPlans, plan)
			usedPartitions[plan.PartitionID] = true
			for _, affectedID := range plan.AffectedPartitions {
				usedPartitions[affectedID] = true
			}
		}
	}

	// 更新plans
	copy(plans, validPlans)
	plans = plans[:len(validPlans)]
}

// executeAdjustment 执行调整
func (lb *LoadBalancer) executeAdjustment(plan *AdjustmentPlan) {
	record := AdjustmentRecord{
		Timestamp:   time.Now(),
		Action:      plan.Action,
		PartitionID: plan.PartitionID,
		BeforeState: lb.partitions[plan.PartitionID],
	}

	// 委托给调整器执行
	err := lb.adjuster.ExecuteAdjustment(plan)

	record.Success = (err == nil)
	record.AfterState = lb.partitions[plan.PartitionID]

	// 记录调整历史
	lb.history = append(lb.history, record)
	lb.lastAdjustment = time.Now()

	// 清理过期历史
	if len(lb.history) > 100 {
		lb.history = lb.history[len(lb.history)-100:]
	}
}

// GetAdjustmentHistory 获取调整历史
func (lb *LoadBalancer) GetAdjustmentHistory() []AdjustmentRecord {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	history := make([]AdjustmentRecord, len(lb.history))
	copy(history, lb.history)
	return history
}

// GetLoadStats 获取负载统计
func (lb *LoadBalancer) GetLoadStats() map[string]interface{} {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	stats := make(map[string]interface{})

	totalPlayers := 0
	totalMessages := float64(0)
	partitionCount := len(lb.partitions)

	overloadedCount := 0
	underloadedCount := 0

	for _, metrics := range lb.partitions {
		totalPlayers += metrics.PlayerCount
		totalMessages += metrics.MessageRate

		if lb.isOverloaded(metrics) {
			overloadedCount++
		}
		if lb.isUnderloaded(metrics) {
			underloadedCount++
		}
	}

	stats["total_partitions"] = partitionCount
	stats["total_players"] = totalPlayers
	stats["avg_players_per_partition"] = float64(totalPlayers) / float64(partitionCount)
	stats["total_message_rate"] = totalMessages
	stats["overloaded_partitions"] = overloadedCount
	stats["underloaded_partitions"] = underloadedCount
	stats["last_adjustment"] = lb.lastAdjustment
	stats["adjustment_count"] = len(lb.history)

	return stats
}
