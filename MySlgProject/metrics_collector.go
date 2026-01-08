package main

import (
	"sync"
	"time"
)

// MetricsCollector 性能指标收集器
type MetricsCollector struct {
	partitionMgr   *PartitionManager
	loadBalancer   *LoadBalancer
	collectionInterval time.Duration
	stopChan       chan struct{}
	mutex          sync.RWMutex
	systemMetrics  *SystemMetrics
	partitionHistory map[string][]*PartitionMetrics // 分区历史指标
}

// SystemMetrics 系统整体指标
type SystemMetrics struct {
	TotalPlayers      int
	TotalEntities     int
	TotalPartitions   int
	AvgMessageRate    float64
	AvgResponseTime   time.Duration
	MemoryUsage       int64
	CpuUsage          float64
	LastUpdate        time.Time
}

// PartitionMetricsSnapshot 分区指标快照
type PartitionMetricsSnapshot struct {
	PartitionID string
	Metrics     *PartitionMetrics
	Timestamp   time.Time
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(partitionMgr *PartitionManager, loadBalancer *LoadBalancer,
	collectionInterval time.Duration) *MetricsCollector {

	return &MetricsCollector{
		partitionMgr:      partitionMgr,
		loadBalancer:      loadBalancer,
		collectionInterval: collectionInterval,
		stopChan:          make(chan struct{}),
		systemMetrics:     &SystemMetrics{},
		partitionHistory:  make(map[string][]*PartitionMetrics),
	}
}

// Start 启动收集器
func (mc *MetricsCollector) Start() {
	go mc.collectionLoop()
}

// Stop 停止收集器
func (mc *MetricsCollector) Stop() {
	close(mc.stopChan)
}

// collectionLoop 收集循环
func (mc *MetricsCollector) collectionLoop() {
	ticker := time.NewTicker(mc.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.collectMetrics()
		case <-mc.stopChan:
			return
		}
	}
}

// collectMetrics 收集指标
func (mc *MetricsCollector) collectMetrics() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// 收集分区指标
	partitionMetrics := mc.loadBalancer.GetAllMetrics()

	// 更新分区历史
	for partitionID, metrics := range partitionMetrics {
		if metrics != nil {
			snapshot := &PartitionMetrics{
				PartitionID:     metrics.PartitionID,
				PlayerCount:     metrics.PlayerCount,
				EntityCount:     metrics.EntityCount,
				MessageRate:     metrics.MessageRate,
				CpuUsage:        metrics.CpuUsage,
				MemoryUsage:     metrics.MemoryUsage,
				AvgResponseTime: metrics.AvgResponseTime,
				LastUpdate:      time.Now(),
			}

			mc.updatePartitionHistory(partitionID, snapshot)
		}
	}

	// 计算系统整体指标
	mc.calculateSystemMetrics(partitionMetrics)
}

// updatePartitionHistory 更新分区历史
func (mc *MetricsCollector) updatePartitionHistory(partitionID string, metrics *PartitionMetrics) {
	history, exists := mc.partitionHistory[partitionID]
	if !exists {
		history = make([]*PartitionMetrics, 0)
	}

	// 添加新指标
	history = append(history, metrics)

	// 保留最近100个数据点
	maxHistory := 100
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}

	mc.partitionHistory[partitionID] = history
}

// calculateSystemMetrics 计算系统指标
func (mc *MetricsCollector) calculateSystemMetrics(partitionMetrics map[string]*PartitionMetrics) {
	totalPlayers := 0
	totalEntities := 0
	totalMessageRate := float64(0)
	totalResponseTime := time.Duration(0)
	validPartitions := 0

	for _, metrics := range partitionMetrics {
		if metrics != nil {
			totalPlayers += metrics.PlayerCount
			totalEntities += metrics.EntityCount
			totalMessageRate += metrics.MessageRate
			totalResponseTime += metrics.AvgResponseTime
			validPartitions++
		}
	}

	mc.systemMetrics.TotalPlayers = totalPlayers
	mc.systemMetrics.TotalEntities = totalEntities
	mc.systemMetrics.TotalPartitions = len(partitionMetrics)

	if validPartitions > 0 {
		mc.systemMetrics.AvgMessageRate = totalMessageRate / float64(validPartitions)
		mc.systemMetrics.AvgResponseTime = totalResponseTime / time.Duration(validPartitions)
	}

	// 这里可以添加实际的内存和CPU监控
	mc.systemMetrics.MemoryUsage = 0 // 需要实际实现
	mc.systemMetrics.CpuUsage = 0.0   // 需要实际实现
	mc.systemMetrics.LastUpdate = time.Now()
}

// GetSystemMetrics 获取系统指标
func (mc *MetricsCollector) GetSystemMetrics() *SystemMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// 返回副本避免并发问题
	metrics := *mc.systemMetrics
	return &metrics
}

// GetPartitionMetrics 获取分区指标
func (mc *MetricsCollector) GetPartitionMetrics(partitionID string) []*PartitionMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	history, exists := mc.partitionHistory[partitionID]
	if !exists {
		return nil
	}

	// 返回副本
	result := make([]*PartitionMetrics, len(history))
	copy(result, history)
	return result
}

// GetPartitionMetricsAtTime 获取指定时间的分区指标
func (mc *MetricsCollector) GetPartitionMetricsAtTime(partitionID string, timestamp time.Time) *PartitionMetrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	history, exists := mc.partitionHistory[partitionID]
	if !exists {
		return nil
	}

	// 找到最接近的时间点
	var closest *PartitionMetrics
	minDiff := time.Hour // 最大时间差

	for _, metrics := range history {
		if diff := timestamp.Sub(metrics.LastUpdate); diff >= 0 && diff < minDiff {
			minDiff = diff
			closest = metrics
		}
	}

	if closest != nil {
		// 返回副本
		result := *closest
		return &result
	}

	return nil
}

// GetPartitionTrend 获取分区趋势分析
func (mc *MetricsCollector) GetPartitionTrend(partitionID string, duration time.Duration) *PartitionTrend {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	history, exists := mc.partitionHistory[partitionID]
	if !exists || len(history) < 2 {
		return nil
	}

	cutoff := time.Now().Add(-duration)
	validHistory := make([]*PartitionMetrics, 0)

	// 筛选时间范围内的数据
	for _, metrics := range history {
		if metrics.LastUpdate.After(cutoff) {
			validHistory = append(validHistory, metrics)
		}
	}

	if len(validHistory) < 2 {
		return nil
	}

	trend := &PartitionTrend{
		PartitionID:    partitionID,
		TimeRange:      duration,
		DataPoints:     len(validHistory),
		PlayerTrend:    mc.calculateTrend(validHistory, func(m *PartitionMetrics) float64 { return float64(m.PlayerCount) }),
		MessageTrend:   mc.calculateTrend(validHistory, func(m *PartitionMetrics) float64 { return m.MessageRate }),
		CpuTrend:       mc.calculateTrend(validHistory, func(m *PartitionMetrics) float64 { return m.CpuUsage }),
		MemoryTrend:    mc.calculateTrend(validHistory, func(m *PartitionMetrics) float64 { return m.MemoryUsage }),
		ResponseTrend:  mc.calculateResponseTimeTrend(validHistory),
	}

	return trend
}

// calculateTrend 计算趋势
func (mc *MetricsCollector) calculateTrend(history []*PartitionMetrics, extractor func(*PartitionMetrics) float64) TrendAnalysis {
	if len(history) < 2 {
		return TrendAnalysis{}
	}

	values := make([]float64, len(history))
	for i, metrics := range history {
		values[i] = extractor(metrics)
	}

	return mc.analyzeTrend(values)
}

// calculateResponseTimeTrend 计算响应时间趋势
func (mc *MetricsCollector) calculateResponseTimeTrend(history []*PartitionMetrics) TrendAnalysis {
	if len(history) < 2 {
		return TrendAnalysis{}
	}

	values := make([]float64, len(history))
	for i, metrics := range history {
		values[i] = float64(metrics.AvgResponseTime.Nanoseconds())
	}

	return mc.analyzeTrend(values)
}

// analyzeTrend 分析趋势
func (mc *MetricsCollector) analyzeTrend(values []float64) TrendAnalysis {
	if len(values) < 2 {
		return TrendAnalysis{}
	}

	// 计算简单线性回归
	n := float64(len(values))
	sumX := n * (n - 1) / 2
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, y := range values {
		x := float64(i)
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	// 斜率
	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// 趋势判断
	var trend TrendDirection
	if slope > 1 { // 上升
		trend = TrendRising
	} else if slope < -1 { // 下降
		trend = TrendFalling
	} else { // 稳定
		trend = TrendStable
	}

	return TrendAnalysis{
		Slope:         slope,
		Direction:     trend,
		CurrentValue:  values[len(values)-1],
		AverageValue:  sumY / n,
		MinValue:      mc.findMin(values),
		MaxValue:      mc.findMax(values),
	}
}

// findMin 查找最小值
func (mc *MetricsCollector) findMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// findMax 查找最大值
func (mc *MetricsCollector) findMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// PartitionTrend 分区趋势
type PartitionTrend struct {
	PartitionID   string
	TimeRange     time.Duration
	DataPoints    int
	PlayerTrend   TrendAnalysis
	MessageTrend  TrendAnalysis
	CpuTrend      TrendAnalysis
	MemoryTrend   TrendAnalysis
	ResponseTrend TrendAnalysis
}

// TrendAnalysis 趋势分析
type TrendAnalysis struct {
	Slope        float64
	Direction    TrendDirection
	CurrentValue float64
	AverageValue float64
	MinValue     float64
	MaxValue     float64
}

// TrendDirection 趋势方向
type TrendDirection int

const (
	TrendRising TrendDirection = iota
	TrendFalling
	TrendStable
)

// GetMetricsSummary 获取指标汇总
func (mc *MetricsCollector) GetMetricsSummary() map[string]interface{} {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	systemMetrics := mc.GetSystemMetrics()

	summary := map[string]interface{}{
		"system": map[string]interface{}{
			"total_players":     systemMetrics.TotalPlayers,
			"total_entities":    systemMetrics.TotalEntities,
			"total_partitions":  systemMetrics.TotalPartitions,
			"avg_message_rate":  systemMetrics.AvgMessageRate,
			"avg_response_time": systemMetrics.AvgResponseTime.String(),
			"memory_usage":      systemMetrics.MemoryUsage,
			"cpu_usage":         systemMetrics.CpuUsage,
			"last_update":       systemMetrics.LastUpdate,
		},
		"partitions": map[string]interface{}{
			"count": len(mc.partitionHistory),
			"active": len(mc.partitionMgr.GetAllPartitions()),
		},
		"collection": map[string]interface{}{
			"interval": mc.collectionInterval.String(),
			"history_points": len(mc.partitionHistory),
		},
	}

	return summary
}
