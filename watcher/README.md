# Typesense Sync Watcher

A robust event processing system that synchronizes MongoDB changes with Typesense, featuring multiple layers of persistence and fault tolerance.

## Overview

The watcher system monitors MongoDB changes and synchronizes them with Typesense, ensuring data consistency across both systems. It implements a multi-layered persistence strategy to guarantee data safety and system reliability.

## Architecture

### 1. Event Processing Pipeline
- **MongoDB Change Stream**: Monitors real-time changes in MongoDB collections
- **Event Buffer**: In-memory buffer for batching events
- **Typesense Sync**: Synchronizes changes with Typesense search engine

### 2. Persistence Layers
1. **Primary Redis Queue**
   - In-memory queue for fast event processing
   - RDB persistence (snapshots)
   - AOF persistence (append-only log)
   - Configurable save intervals

2. **Backup Redis Queue**
   - Redundant queue for high availability
   - Independent persistence configuration
   - Automatic failover support

3. **MongoDB Fallback Store**
   - Last-resort persistence layer
   - Stores events when Redis queues are unavailable
   - Maintains event order and status

## Features

### Event Processing
- Real-time MongoDB change monitoring
- Batch processing for efficiency
- Configurable buffer sizes and flush intervals
- Support for insert, update, and delete operations

### Fault Tolerance
- Multiple persistence layers
- Automatic failover between Redis instances
- Retry mechanisms with exponential backoff
- Connection pooling for better performance

### Data Safety
- Redis RDB snapshots
- Redis AOF persistence
- MongoDB fallback storage
- Event status tracking

### Monitoring
- Queue health checks
- Event processing metrics
- Error logging and tracking
- Connection status monitoring

## Configuration

### Environment Variables

#### MongoDB Configuration
```
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=neatlisting
MONGODB_COLLECTION=listings
```

#### Typesense Configuration
```
TYPESENSE_API_KEY=xyz
TYPESENSE_HOST=localhost
TYPESENSE_PORT=8108
TYPESENSE_PROTOCOL=http
TYPESENSE_COLLECTION_NAME=listings
TYPESENSE_CONNECTION_TIMEOUT=5s
TYPESENSE_KEEPALIVE_TIMEOUT=30s
TYPESENSE_MAX_RETRIES=3
TYPESENSE_RETRY_BACKOFF=100ms
```

#### Buffer Configuration
```
MAX_BUFFER_SIZE=1000
BATCH_SIZE=100
FLUSH_INTERVAL=5
```

#### Redis Configuration
```
# Primary Redis
PRIMARY_REDIS_HOST=localhost
PRIMARY_REDIS_PORT=6379
PRIMARY_REDIS_PASSWORD=
PRIMARY_REDIS_DB=0
PRIMARY_REDIS_QUEUE_KEY=typesense_sync_queue
PRIMARY_REDIS_POOL_SIZE=10
PRIMARY_REDIS_MIN_IDLE_CONNS=5
PRIMARY_REDIS_MAX_RETRIES=3
PRIMARY_REDIS_RETRY_BACKOFF=100ms
PRIMARY_REDIS_CONN_MAX_LIFETIME=1h
PRIMARY_REDIS_SAVE_INTERVAL=300s
PRIMARY_REDIS_APPEND_ONLY=true
PRIMARY_REDIS_APPEND_FILENAME=appendonly.aof
PRIMARY_REDIS_RDB_FILENAME=dump.rdb

# Backup Redis
BACKUP_REDIS_HOST=localhost
BACKUP_REDIS_PORT=6380
BACKUP_REDIS_PASSWORD=
BACKUP_REDIS_DB=0
BACKUP_REDIS_QUEUE_KEY=typesense_sync_queue_backup
BACKUP_REDIS_POOL_SIZE=10
BACKUP_REDIS_MIN_IDLE_CONNS=5
BACKUP_REDIS_MAX_RETRIES=3
BACKUP_REDIS_RETRY_BACKOFF=100ms
BACKUP_REDIS_CONN_MAX_LIFETIME=1h
BACKUP_REDIS_SAVE_INTERVAL=300s
BACKUP_REDIS_APPEND_ONLY=true
BACKUP_REDIS_APPEND_FILENAME=appendonly_backup.aof
BACKUP_REDIS_RDB_FILENAME=dump_backup.rdb
```

## Operation Flow

1. **Event Capture**
   - MongoDB change stream detects changes
   - Events are added to memory buffer
   - Events are persisted to Redis queues

2. **Event Processing**
   - Buffer is flushed based on size or time
   - Events are processed in batches
   - Changes are synchronized with Typesense

3. **Persistence Strategy**
   - Primary Redis queue is used first
   - Backup Redis queue is used if primary fails
   - MongoDB fallback is used if both Redis queues fail

4. **Recovery Process**
   - System recovers events from Redis queues
   - If Redis queues are unavailable, recovers from MongoDB
   - Maintains event order during recovery

## Error Handling

- **Connection Errors**: Automatic retry with exponential backoff
- **Processing Errors**: Events are stored in fallback
- **Queue Errors**: Automatic failover to backup queue
- **Typesense Errors**: Events are preserved for retry

## Performance Considerations

- **Connection Pooling**: Optimized Redis and MongoDB connections
- **Batch Processing**: Efficient event processing in batches
- **Memory Management**: Configurable buffer sizes
- **Persistence Tuning**: Configurable save intervals

## Monitoring and Maintenance

- **Health Checks**: Regular queue health monitoring
- **Error Logging**: Comprehensive error tracking
- **Performance Metrics**: Queue lengths and processing times
- **Recovery Status**: Event recovery tracking

## Development vs Production

### Development Environment
- Less frequent persistence
- Smaller buffer sizes
- Shorter timeouts
- Local service connections

### Production Environment
- More frequent persistence
- Larger buffer sizes
- Longer timeouts
- Secure service connections
- Higher connection pool sizes

## Building and Running

1. **Build the watcher**:
   ```bash
   cd watcher
   go build -o watcher
   ```

2. **Run the watcher**:
   ```bash
   ./watcher
   ```

3. **Environment Setup**:
   - Copy `env.dev` to `.env` for development
   - Copy `env.prod` to `.env` for production
   - Adjust settings as needed

## Dependencies

- MongoDB Go Driver
- Redis Go Client
- Typesense Go Client
- Godotenv for configuration

## Future Enhancements

1. **Metrics and Monitoring**
   - Prometheus metrics integration
   - Grafana dashboards
   - Alerting system

2. **Advanced Features**
   - Event filtering
   - Custom event transformations
   - Rate limiting
   - Circuit breaker pattern

3. **Operational Improvements**
   - Automated backup rotation
   - Health check API
   - Graceful shutdown
   - Hot reloading of configuration 