package persistence

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/go-redis/redis/v8"
	"demo/internal/protocol"
)

// RedisWriterActor handles asynchronous Redis persistence
type RedisWriterActor struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisWriterActor creates a new RedisWriterActor
func NewRedisWriterActor(redisAddr string) actor.Actor {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	ctx := context.Background()

	return &RedisWriterActor{
		client: rdb,
		ctx:    ctx,
	}
}

// Receive handles persistence requests
func (rw *RedisWriterActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *protocol.PersistRequest:
		rw.handlePersistRequest(msg)
	default:
		log.Printf("RedisWriter received unknown message: %T", msg)
	}
}

// handlePersistRequest processes a persistence request
func (rw *RedisWriterActor) handlePersistRequest(req *protocol.PersistRequest) {
	go rw.persistAsync(req) // Async to avoid blocking actor
}

// persistAsync performs the actual persistence
func (rw *RedisWriterActor) persistAsync(req *protocol.PersistRequest) {
	// Retry logic
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := rw.doPersist(req)
		if err == nil {
			log.Printf("Successfully persisted OpID %s", req.OpID)
			return
		}
		log.Printf("Persist failed for OpID %s, retry %d: %v", req.OpID, i+1, err)
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	log.Printf("Persist failed permanently for OpID %s", req.OpID)
}

// doPersist performs the Redis operations
func (rw *RedisWriterActor) doPersist(req *protocol.PersistRequest) error {
	// Partition snapshot
	if req.Snapshot != nil {
		key := "partition:" + req.PartitionID + ":snapshot"
		err := rw.client.Set(rw.ctx, key, req.Snapshot, 0).Err()
		if err != nil {
			return err
		}
	}

	// Entity data
	if req.EntityData != nil && req.EntityID != "" {
		key := "entity:" + req.EntityID
		err := rw.client.HMSet(rw.ctx, key, map[string]interface{}{
			"data": req.EntityData,
			"updated": time.Now().Unix(),
		}).Err()
		if err != nil {
			return err
		}
	}

	// Event log
	if req.EventLog != nil {
		key := "partition:" + req.PartitionID + ":events"
		err := rw.client.LPush(rw.ctx, key, req.EventLog).Err()
		if err != nil {
			return err
		}
		// Trim to keep only recent events
		rw.client.LTrim(rw.ctx, key, 0, 999)
	}

	// OpID dedup marker (TTL)
	if req.OpID != "" {
		key := "op:" + req.OpID
		err := rw.client.Set(rw.ctx, key, "1", time.Minute*5).Err() // 5 min TTL
		if err != nil {
			return err
		}
	}

	return nil
}

// BatchPersist could be added for batching multiple requests
func (rw *RedisWriterActor) BatchPersist(requests []*protocol.PersistRequest) {
	// Implement batching logic if needed
	for _, req := range requests {
		rw.persistAsync(req)
	}
}
