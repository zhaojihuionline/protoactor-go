package main

import (
	"time"
)

// Config 游戏配置
type Config struct {
	World WorldConfig
	AOI   AOIConfig
	Partition PartitionConfig
	Performance PerformanceConfig
}

// WorldConfig 世界配置
type WorldConfig struct {
	Width  int // 地图宽度
	Height int // 地图高度
}

// AOIConfig AOI配置
type AOIConfig struct {
	ViewRange       int           // 视野范围（格子数）
	UpdateInterval  time.Duration // AOI更新间隔
	BatchSize       int           // 批量处理大小
	CacheExpiration time.Duration // 缓存过期时间
}

// PartitionConfig 分区配置
type PartitionConfig struct {
	DefaultSize       int           // 默认分区大小
	MinSize           int           // 最小分区大小
	MaxSize           int           // 最大分区大小
	MonitorInterval   time.Duration // 监控间隔
	AdjustmentCooldown time.Duration // 调整冷却时间
	LoadThresholds    LoadThresholds
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	MaxMessageQueueSize int
	WorkerPoolSize      int
	StateSyncBatchSize  int
	MetricsCollectInterval time.Duration
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		World: WorldConfig{
			Width:  1200,
			Height: 1200,
		},
		AOI: AOIConfig{
			ViewRange:       50,
			UpdateInterval:  time.Second * 1,
			BatchSize:       100,
			CacheExpiration: time.Minute * 5,
		},
		Partition: PartitionConfig{
			DefaultSize:        100,
			MinSize:           50,
			MaxSize:           200,
			MonitorInterval:   time.Second * 30,
			AdjustmentCooldown: time.Minute * 5,
			LoadThresholds: LoadThresholds{
				MaxPlayersPerPartition: 500,
				MinPlayersPerPartition: 20,
				MaxMessageRate:         10000,
				MinMessageRate:         100,
				MaxCpuUsage:            0.8,
				MinCpuUsage:            0.1,
				MaxMemoryUsage:         0.85,
				AdjustmentCooldown:     time.Minute * 5,
			},
		},
		Performance: PerformanceConfig{
			MaxMessageQueueSize: 10000,
			WorkerPoolSize:      10,
			StateSyncBatchSize:  50,
			MetricsCollectInterval: time.Second * 10,
		},
	}
}

// LoadConfig 加载配置（可从文件或环境变量加载）
func LoadConfig() *Config {
	return DefaultConfig()
}
