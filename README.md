# WatchParty - Video Sync Platform

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
Build Status, ...

WatchParty is a distributed video synchronization platform that allows users to watch content together in real-time across multiple regions. It is built on a scalable, event-driven architecture designed for high availability and low latency.

## Technology Stack

### Backend Services
* **Runtime:** Go 1.24 - High-performance, concurrent backend services
* **Web Framework:** Gorilla Mux - HTTP routing and middleware
* **Real-time Communication:** Gorilla WebSocket - Low-latency bidirectional communication
* **Authentication:** JWT with refresh tokens, bcrypt password hashing

### Data & Storage
* **Primary Database:** PostgreSQL 17
* **Caching & Session Store:** Redis Cluster - High-performance in-memory data structure store
* **Audit & Compliance:** ImmuDB - Immutable for audit trails
* **Media Storage:** Google Cloud Storage - Scalable object storage for video files
* **Message Queue:** Google Cloud Pub/Sub - Global event distribution for cross-region sync

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

### Implementation: Sending Emails

* **Google Workspace account's Gmail SMTP server** - this is reliable and requires no extra cost.
   lorem ipzum

* **Future Scalable Option** - migrate to a dedicated transactional email service like **SendGrid**, **Mailgun**, or **Postmark** for better out of the box deliverability and analytics.
