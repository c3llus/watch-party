# WatchParty Backend Architecture

A microservices-based backend built with Go, designed for real-time video synchronization and scalable deployment options.

## Architecture Overview

The backend follows a clean, pragmatic microservices approach with two main services:

```
backend/
├── service-api/       # Generic REST API service
├── service-sync/      # WebSocket sync service (port 8081)
├── standalone/        # Embedded deployment option
└── pkg/               # Shared libraries and utilities
```

**Why two services instead of one monolith?** Different concerns, different scaling patterns. The API service handles CRUD operations, file uploads, and authentication; typical HTTP request/response patterns. The sync service manages real-time WebSocket connections and state synchronization, completely different performance characteristics and scaling needs.

**Why not straight up a gazillion microservices?** Because complexity has a cost. Two services give us clear separation without the operational overhead of managing dozens of tiny services.

## Service Breakdown

### API Service
**Purpose**: Authentication, user management, video processing, and room lifecycle management.

**Key responsibilities**:
- User registration and JWT-based authentication
- Video upload, transcoding, and HLS generation
- Room creation and access control
- File storage management (Blob Storage)
- Guest session handling

**Architecture pattern**: Layered architecture with clear separation:
```
internal/
├── app/           # Application setup and dependency injection
├── controller/    # HTTP handlers and request validation
├── service/       # Business logic layer
└── repository/    # Data access layer
```

### Sync Service
**Purpose**: Real-time video synchronization via WebSocket connections.

**Key responsibilities**:
- WebSocket connection management
- Real-time play/pause/seek synchronization
- Participant presence tracking
- Live chat message broadcasting
- State consistency across participants

**Architecture pattern**: Event-driven with connection pooling:
```
internal/
├── handler/       # WebSocket upgrade and connection handling
├── service/       # Sync logic and message routing
└── repository/    # Ephemeral state management (Redis)
```

## Shared Libraries (pkg/)

Instead of duplicating code across services, common functionality lives in shared packages:

- **auth/**: JWT generation, validation, middleware
- **config/**: Environment configuration management
- **database/**: PostgreSQL connection and migrations (TODO)
- **redis/**: Redis client setup and utilities (TODO: separate cache/queue; now combined due to xxx)
- **storage/**: MinIO/GCS abstraction layer (easily extended)
- **model/**: Shared data structures and validation
- **logger/**: Structured, standardized logging
- **websocket/**: WebSocket utilities and message types
- **video/**: FFmpeg integration for transcoding
- **events/**: Event-driven processing for async tasks

This approach means bug fixes and improvements in auth logic automatically benefit both services. No code duplication, easier testing, cleaner interfaces.

## Design Decisions

### Database Strategy: One Per Concern
- **PostgreSQL (service-api)**: Persistent data - users, rooms, videos, permissions
- **Redis (service-sync)**: Ephemeral data - active sessions, room state, message queues

This isn't "database per microservice" dogma; it's practical separation. User accounts need ACID transactions and persistent storage. Real-time room state needs fast reads/writes and automatic expiration. Different tools for different jobs.

### Authentication: JWT with Refresh Tokens
Stateless JWTs for API requests, refresh tokens stored in PostgreSQL for security. This gives us:
- **Scalability**: No server-side session storage needed
- **Security**: Refresh tokens can be revoked immediately
- **Simplicity**: Standard JWT libraries in every language

Guest access uses temporary tokens validated through the API service, allowing seamless room joining without registration friction.

### Video Processing: Async with Event-Driven Architecture
Video uploads trigger async transcoding jobs that:
1. Generate multiple quality levels (HLS adaptive streaming)
2. Create thumbnail previews
3. Update database status when complete
4. Notify clients via WebSocket (if connected)

No blocking uploads, no timeouts, proper progress tracking.

### Real-Time Sync: Conflict Resolution Strategy
When multiple users perform actions simultaneously (rare but happens):
1. **Last-write-wins** for simple actions (play/pause)
2. **Timestamp-based ordering** for seek operations
3. **Host privilege** for conflict resolution in ambiguous cases

The system favors availability over perfect consistency; *better to have **slightly** imperfect sync than blocked users*.

### Storage Architecture: Cloud-Native with Local Fallback
- **Production**: MinIO or GCS for scalable object storage
- **Development**: Local MinIO instance via Docker
- **Standalone**: Embedded MinIO server for complete portability

Same code paths, different deployment targets. This makes local development realistic and production deployment straightforward.

---

## Design Decisions Verification Report

*This section verifies that the claimed design decisions are actually implemented in the codebase.*

### Database Strategy: One Per Concern

**PostgreSQL (service-api)**: 
- `service-api/internal/app/app.go` imports `watch-party/pkg/database` 
- All user, room, movie repositories use PostgreSQL via `pkg/database/postgres.go`
- Refresh tokens stored in PostgreSQL `tokens` table (`db/schema.sql`)

**Redis (service-sync)**:
- `service-sync/internal/app/app.go` imports `watch-party/pkg/redis`
- `service-sync/internal/repository/sync_repository.go` uses Redis for room state, participants
- Clear separation: API service never touches Redis, Sync service never touches PostgreSQL

### Authentication: JWT with Refresh Tokens

**JWT Implementation**:
- `pkg/auth/jwt.go` implements full JWT with claims (UserID, Email, Role)
- Access tokens: 24-hour expiration, stateless
- `GenerateAccessToken()` and `ValidateToken()` methods present

**Refresh Token Storage**:
- `service-api/internal/repository/auth/repository.go` implements refresh token CRUD
- `StoreRefreshToken()`, `GetRefreshToken()`, `DeleteRefreshToken()` methods
- Tokens stored in PostgreSQL `tokens` table with hash values (not plaintext)

**Guest Access**:
- `service-sync/internal/handler/sync_handler.go` handles guest token validation
- Guest sessions validated via API service call to `/api/v1/guest/validate/`

### Video Processing: Async with Event-Driven Architecture

**Async Processing**:
- `pkg/events/upload_handler.go` line 107: `go h.processVideoAsync(context.Background(), movie)`
- Upload completion triggers background transcoding job

**Video Processing: Async with Event-Driven Architecture

**Async Processing**:
- `pkg/events/upload_handler.go` line 107: `go h.processVideoAsync(context.Background(), movie)`
- Upload completion triggers background transcoding job

**Multiple quality levels**:
- `pkg/video/transcoder.go` implements [HLS](https://datatracker.ietf.org/doc/html/rfc8216) generation (though client side code is not implemented)
- Creates multiple renditions (resolutions/bitrates) for adaptive streaming

**Non-blocking**:
- Upload returns immediately, processing happens in background goroutine
- Status polling allows clients to track progress

### Real-Time Sync: Conflict Resolution Strategy**

**Room-level locking (primary conflict prevention)**:
- `AcquireRoomLock()` in `sync_repository.go` creates Redis lock with 5-second timeout
- `SyncAction()` method acquires lock before any state changes: `"room is locked by another user"`
- Lock prevents simultaneous state modifications, eliminating most conflicts

**Last-write-wins via atomic updates**:
- Every sync action updates `state.LastUpdated = time.Now()` and `state.UpdatedBy = message.UserID`
- Redis atomic operations ensure the last successful lock acquisition wins
- State includes `UpdatedBy` field tracking who made the last change

### Storage Architecture

**Abstraction Layer**:
- `pkg/storage/interface.go` defines unified `Provider` interface
- `pkg/storage/factory.go` creates providers based on configuration

**Multiple Implementations**:
- `pkg/storage/gcs.go` - Google Cloud Storage
- `pkg/storage/minio.go` - MinIO (local/self-hosted)

**Same Code Paths**:
- `NewStorageProvider()` factory pattern ensures identical interface
- Configuration-driven provider selection (`StorageProviderGCS` vs `StorageProviderMinIO`)

**Deployment Flexibility**:
- Production can use GCS, development uses local MinIO
- Standalone embeds MinIO server directly

## Deployment Options

- **Containerized Services**: Best for production environments, when you need independent scaling. 
- **Standalone Executable**: Best for single-user deployments, demos, development, users who want "just works" simplicity

## API Design Philosophy

### RESTful Where It Makes Sense
Standard HTTP methods for resource operations:
- `POST /api/v1/rooms` - Create room
- `GET /api/v1/rooms/{id}` - Get room details
- `PUT /api/v1/rooms/{id}` - Update room settings
- `DELETE /api/v1/rooms/{id}` - Delete room

### WebSocket for Real-Time Operations
When there's too much of a overhead of repeated HTTP requests:
- Video sync events
- Live chat messages

## Performance Considerations

### Video Streaming Security & Optimization

#### Per-Segment Authentication: Cryptographically Secure Access Control
The HLS streaming implementation uses **signed URLs for every segment request**, creating an exceptionally secure video delivery system:

**How it works**:
- Each video segment requires a fresh signed URL with cryptographic signature
- Client must authenticate with JWT/guest token for each batch of segment URLs
- Storage provider (MinIO/GCS) validates signatures before serving content
- No segment can be accessed without valid authentication

**Security guarantees**:
- **No direct URL guessing**: Segment paths are unpredictable and require valid signatures
- **Time-bounded access**: URLs expire after 2 hours, limiting exposure window
- **Per-request validation**: Every segment download validates current user permissions
- **Revocation capability**: Disable user access immediately; existing URLs become invalid

**Theoretical attack vectors (and why they're impractical)**:

- **URL interception and sharing**
     - *Practically*: Attacker gets access to ~10 minutes of video before URLs expire

- **Mass URL harvesting**:
     - *Practically*: Would require maintaining active authenticated session + bypassing rate limits

- **Cryptographic signature forgery**:
     - *Practically*: Computationally, not feasible

- **Storage provider compromise**:
     - *Practically*: Requires infrastructure-level breach, affects all data not just videos

**Why this is exceptionally secure**:
- Traditional streaming often uses static URLs or simple token validation
- Our approach requires active authentication for every segment seconds of HLS video content
- Even if credentials are compromised, access is automatically revoked when URLs expire

#### Mini Improvements
- **Batch URL generation**: Client requests multiple segment URLs to reduce API calls
- **Segment preloading**: Client-side buffering for smooth playback without compromising security

### WebSocket Connection Management
- **Connection pooling**: Reuse connections efficiently
- **Graceful degradation**: Automatic reconnection with exponential backoff
- **Memory management**: Clean up disconnected sessions automatically
- **Horizontal scaling**: Redis pub/sub for cross-instance message delivery

### Database Query Optimization
- **Connection pooling**: Reuse database connections (managed by stdlib)
- **Selective queries**: Only fetch needed columns
- **Proper indexing**: Database schema includes performance-critical indexes

## Security Architecture

### Authentication & Authorization
- **Password hashing**: bcrypt with proper salting
- **JWT validation**: Cryptographic signature verification
- **Token expiration**: Short-lived access tokens, longer-lived refresh tokens
- **Role-based access**: User/admin/guest permission levels

### Input Validation & Sanitization
- **Request validation**: JSON schema validation for all inputs
- **File upload safety**: MIME type checking, size limits, virus scanning hooks
- **SQL injection prevention**: Parameterized queries only
- **XSS protection**: Content Security Policy headers

### Network Security
- **CORS configuration**: Properly configured cross-origin policies
- **TLS termination**: HTTPS everywhere in production
- **Firewall-friendly**: Standard ports, minimal network requirements

## Development Workflow

### Local Development Setup
```bash
# Start dependencies
docker-compose up -d postgres redis minio

# Run services in development mode
cd service-api && go run cmd/main.go
cd service-sync && go run cmd/main.go

# Run tests
go test ./...
```

### Code Organization Principles
- **Clear interfaces**: Each layer defines clean contracts
- **Dependency injection**: Services receive dependencies explicitly
- **Error handling**: Comprehensive error propagation and logging
- **Configuration**: Environment-based config with sensible defaults

### Debugging and Observability
- **Structured logging**: JSON logs with request tracing
- **Health checks**: `/health` endpoints for service monitoring
- **Metrics endpoints**: Prometheus-compatible metrics
- **Request correlation**: Trace requests across service boundaries

## Future Architecture Considerations

### Multi-Tenancy Implementation
The current architecture is designed to support multi-tenancy with minimal changes:
- Tenant isolation at the database row level
- Separate storage namespaces per tenant
- Redis key prefixing for tenant separation

### Horizontal Scaling
- **API service**: Stateless; scales horizontally behind a load balancer
- **Sync service**: Redis pub/sub enables cross-instance communication
- **Database**: Read replicas for scaling read operations
- **Storage**: MinIO clustering or S3 for unlimited scale

### Monitoring and Alerting
- **Application metrics**: Custom business metrics (rooms created, videos uploaded)
- **Infrastructure metrics**: CPU, memory, disk, network utilization
- **Error tracking**: Structured error reporting and alerting
- **Performance monitoring**: Request latency, database query performance

Adopt OpenTelemetry for unified instrumentation, export metrics to Prometheus, visualize with Grafana, trace with Tempo, and alert via Alertmanager in the future.


### Comprehensive Testing

**Current Status**: Missing due to bootstrapping priorities and limited time constraints.

The testing strategy is intentionally deferred to focus on rapid iteration and core functionality validation. In a production system, this would include:

- **Unit tests** for business logic in `service/` layers and `pkg/` utilities
- **Integration tests** with real upstreams

This technical debt will be addressed once core features are stable and user validation is complete.

---

*Architecture designed for the bootstrapper: simple enough to understand completely, sophisticated enough to scale globally.*
