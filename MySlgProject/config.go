package main

import (
	"time"
)

// Config 游戏配置
type Config struct {
	World WorldConfig
	AOI   AOIConfig
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


// PerformanceConfig 性能配置
type PerformanceConfig struct {
	MaxMessageQueueSize int
	WorkerPoolSize      int
	StateSyncBatchSize  int
	MetricsCollectInterval time.Duration
}

// 分区常量配置
const (
	PartitionSize     = 100 // 每个分区的大小
	PartitionsPerRow  = 12  // 每行分区数 (1200/100)
	TotalPartitions   = 144 // 总分区数 (12*12)
)

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
