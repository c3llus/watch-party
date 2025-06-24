# WatchParty - Video Sync Platform

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
Build Status, ...

WatchParty is a distributed video synchronization platform that allows users to watch content together in real-time across multiple regions. It is built on a scalable, event-driven architecture designed for high availability and low latency.

## Technology Stack

### Backend Services
* **Runtime:** Go 1.24 - High-performance, concurrent backend services
* **Web Framework:** Gin - HTTP routing and middleware for REST API
* **Real-time Communication:** Gorilla WebSocket - Low-latency bidirectional communication
* **Authentication:** JWT with refresh tokens, bcrypt password hashing
* **Service Architecture:** Microservices with clean architecture pattern
  - **service-api**: RESTful API for user management, rooms, movies
  - **service-sync**: WebSocket-based real-time synchronization service

### Data & Storage
* **Primary Database:** PostgreSQL 17
* **Caching & Session Store:** Redis - High-performance in-memory data structure store for caching and pub/sub messaging
* **Audit & Compliance:** ImmuDB - Immutable for audit trails
* **Media Storage:** Google Cloud Storage - Scalable object storage for video files
* **Message Queue:** Redis Pub/Sub - Real-time message broadcasting for WebSocket synchronization

### Frontend
TBD

### Infrastructure
TBD

### Monitoring & Observability
TBD
otel, jaeger, prom, cloud monitoring?, cloud logigng?, alerts

### Development & Deployment
* **Version Control:** Git with GitHub
* **CI/CD:** GitHub Actions
* **Code Quality:** golangci-lint, gofmt, go vet
* **Environment Management:** Docker Compose for local development

## Architectural & Implementation Notes

### Service Architecture

The WatchParty platform consists of two main microservices:

#### service-api (Port 8080)
- **User Management**: Registration, authentication, profile management
- **Room Management**: Create rooms, invite users, manage access
- **Movie Management**: Upload, store, and serve video content
- **RESTful API**: JSON-based HTTP endpoints for frontend integration

#### service-sync (Port 8081)
- **Real-time Synchronization**: WebSocket-based video playback synchronization
- **Sub-millisecond Sync**: Near real-time play/pause/seek actions across participants
- **Participant Management**: Track active users and their playback state
- **Redis Pub/Sub**: Scalable message broadcasting for multi-instance deployments

### Real-time Synchronization Features

- **Play/Pause Sync**: Synchronized playback control across all participants
- **Seek Synchronization**: Jump to specific timestamps with sub-second precision
- **Buffering Management**: Handle buffering states and wait for all participants
- **Participant Tracking**: Real-time participant list with connection status
- **Heartbeat Monitoring**: Automatic detection of disconnected participants
- **State Persistence**: Room state cached in Redis with PostgreSQL fallback

### Implementation: Sending Emails

* **Google Workspace account's Gmail SMTP server** - this is reliable and requires no extra cost.
   lorem ipzum

* **Future Scalable Option** - migrate to a dedicated transactional email service like **SendGrid**, **Mailgun**, or **Postmark** for better out of the box deliverability and analytics.
