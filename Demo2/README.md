# Demo: Actor-based Large Map System

This demo implements a distributed large map system using the actor model with ProtoActor-Go. The system supports a 1200x1200 grid divided into 144 static partitions (12x12), each managed by an independent PartitionActor.

## Architecture

### Core Components

- **PartitionActor**: Manages a 100x100 grid partition, handles entity movements, grid changes, and computes AOI updates.
- **PlayerActor**: Manages individual player state and commands.
- **SessionActor**: Handles client connections, rate limiting, and network I/O simulation.
- **EventRouterActor**: Routes events to appropriate partition actors based on position.
- **RedisWriterActor**: Handles asynchronous persistence to Redis.
- **AOIActor**: Computes Area of Interest updates (optional, can be integrated into PartitionActor).
- **PartitionManager**: Manages the creation and lookup of partition actors.

### Key Features

- **Event-Driven**: No global tick; all state changes triggered by events.
- **Static Partitions**: 144 partitions (12x12 grid), each handling 100x100 cells.
- **Concurrent Isolation**: Each partition operates independently.
- **Redis Persistence**: Asynchronous snapshots and event logging.
- **OpID Deduplication**: Prevents duplicate operations.
- **AOI Updates**: Real-time visibility calculations.
- **Player Passivation**: Inactive players are saved and actors stopped to save memory.

## Prerequisites

- Go 1.19+
- Redis server running on localhost:6379
- ProtoActor-Go dependencies (included in go.mod)

## Quick Start

1. **Start Redis** (if not already running):
   ```bash
   redis-server
   ```

2. **Build and run the server**:
   ```bash
   cd Demo
   go build -o server ./cmd/server
   ./server
   ```

3. **In another terminal, run the CLI client**:
   ```bash
   go build -o client ./cmd/client
   ./client
   ```

4. **Use the CLI**:
   ```
   move 100 100    # Move to position (100,100)
   interact npc_1  # Interact with NPC
   quit            # Exit
   ```

## Running Tests

### Integration Tests

```bash
cd Demo
go test ./cmd/test/...
```

### Benchmarks

```bash
cd Demo
go test -bench=. ./cmd/test/
```

## Configuration

The system uses static configuration:
- Map size: 1200x1200
- Partition size: 100x100
- Grid partitions: 12x12 = 144
- Redis address: 127.0.0.1:6379
- Player passivation: 30 minutes inactivity

## Design Decisions

### Event-Driven Architecture
- Eliminates global synchronization points
- Enables better scalability and fault isolation
- Reduces CPU usage compared to tick-based systems

### Static Partitioning
- Simplifies routing and load balancing
- Avoids complex dynamic partitioning logic
- Suitable for most game scenarios

### Redis for Persistence
- Fast key-value operations
- Supports atomic operations and pub/sub if needed
- Easy to scale horizontally

### Actor Isolation
- Each partition processes events serially
- Prevents race conditions
- Enables parallel processing across partitions

## Extension Points

### Adding New Entity Types
1. Extend `protocol.EntityType`
2. Add handling in `PartitionActor.Receive()`
3. Implement entity-specific logic

### Cross-Partition Operations
- For operations requiring consistency across partitions, implement Saga patterns
- Use distributed transactions or eventual consistency based on requirements

### Network Layer
- Replace `SessionActor` with real network implementation (WebSocket, TCP)
- Add authentication and encryption

### Scaling
- Add cluster support with actor remoting
- Implement load balancing across multiple nodes
- Add partition migration for dynamic scaling

### Monitoring
- Integrate Prometheus metrics
- Add distributed tracing
- Implement health checks

## Performance Considerations

- **Memory**: Each PartitionActor holds 100x100 grid state (~10KB per partition)
- **CPU**: Event processing is lightweight; AOI computation scales with entity density
- **Network**: AOI updates sent per player; consider delta compression
- **Persistence**: Async Redis writes; batch operations for efficiency

## Troubleshooting

### Common Issues

1. **Redis Connection Failed**
   - Ensure Redis is running on 127.0.0.1:6379
   - Check firewall settings

2. **High Memory Usage**
   - Monitor actor count; implement proper passivation
   - Check for message backlog in actor mailboxes

3. **Slow AOI Updates**
   - Profile AOI computation; consider spatial indexing optimizations
   - Implement AOI caching and incremental updates

### Debugging
- Enable actor system logging
- Use `go tool pprof` for performance profiling
- Monitor Redis operations with `redis-cli monitor`

## Future Enhancements

- Implement dynamic AOI with variable view radii
- Add pathfinding and navigation meshes
- Support for large-scale events (battles, sieges)
- Implement player guilds and social features
- Add marketplace and economy systems

## License

This demo is part of the protoactor-go project.
