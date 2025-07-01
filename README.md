# WatchParty - Video Sync Platform

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)

## What's This About?
WatchParty is real-time video synchronization platform that lets groups watch videos together! Think of it as creating a virtual living room where everyone stays perfectly in sync.

- **Perfect video synchronization** across all participants
- **Intuitive room-based experience** - create a room, share the link, start watching
- **Real-time live chat** during viewing sessions
- **Guest access** - no registration required for viewers
- **Self-hosted option** - run it on your own server or locally

Whether you're hosting a movie night with friends, conducting training sessions, or running community watch parties, this platform handles the technical complexity while keeping the experience simple.

## Project Structure & Bootstrapping Philosophy

This project embodies a bootstrapper's approach: everything you need in one repository, organized for efficiency and rapid iteration.

```
watch-party/
├── backend/           # Go microservices architecture
│   ├── service-api/   # Authentication, user management, video handling
│   ├── service-sync/  # Real-time WebSocket synchronization
│   ├── standalone/    # All-in-one executable for easy deployment
│   └── pkg/           # Shared libraries (auth, database, storage, etc.)
├── frontend/          # React + TypeScript web application
├── infrastructure/    # Terraform + Ansible for production deployment
└── docs/              # Architecture diagrams and technical documentation
```

**Why this structure?** As a solo developer or small team, you need to move fast without losing organization. Instead of managing multiple repositories with complex coordination, everything lives together. Backend changes can be tested immediately with frontend updates. Infrastructure changes are versioned alongside the code they deploy. Documentation stays current because it's right there.

This approach scales from "running on my laptop" to "production infrastructure" without restructuring your entire development workflow.

# Core Features

### Real-Time Video Synchronization
Every play, pause, and seek action is instantly synchronized across all participants. Built on WebSocket connections with smart conflict resolution and automatic recovery from network hiccups.

### Meeting-Room Style Experience
Create a room, get a shareable link, and participants join instantly. No complex setup or configuration - just click and watch together. Room hosts have control, but can delegate permissions as needed.

### Live Chat Integration
Built-in chat that runs alongside the video. Participants can react, discuss, and share thoughts in real-time without disrupting the viewing experience.

### Guest Access System
Viewers can join rooms without creating accounts. Perfect for spontaneous movie nights or public events where you don't want registration friction.

### Multi-Platform Support
Works on any device with a modern web browser. Responsive design adapts from mobile phones to large desktop displays.

### HLS Video Streaming
Professional-grade adaptive bitrate streaming. Videos are automatically transcoded to multiple quality levels for smooth playback regardless of connection speed.

## Technology Stack

**Backend Services (Go)**
- `service-api`: User authentication, video processing, room management
- `service-sync`: WebSocket-based real-time synchronization
- Clean architecture with shared packages for common functionality

**Frontend (React + TypeScript)**
- Modern React with hooks and functional components
- TypeScript for type safety and better developer experience
- Custom HLS player

**Infrastructure**
- **Database**: PostgreSQL for persistent data
- **Cache/Message Queue**: Redis for sessions, real-time state, and message queuing
- **Storage**: Google Cloud Storage for files
- **Secrets**: Google Secret Manager
- **Deployment**: Docker containers with Terraform/Ansible automation

## Architecture Philosophy

This platform is designed with a *bootstrapper's mindset* - build something that works well now and can scale later without fundamental rewrites.

**Microservices, but not micro-complexity**: Two focused services instead of dozens. Each handles a clear domain (API vs real-time sync) but they're not so granular that development becomes unwieldy.

**Database-per-service where it matters**: The sync service uses Redis for ephemeral state, while the API service uses PostgreSQL for persistent data. Clear separation without over-engineering.

**Embedded deployment option**: The standalone executable proves the architecture works. If you can embed everything in one binary, your service boundaries are probably right.

**Infrastructure as code from day one**: Even if you're starting small, having repeatable deployments saves time and reduces errors as you grow.

## Toughts - Monetization Strategy

### Freemium Self-Hosted
- **Free**: Single executable, self-hosted deployment
- **Paid Add-ons**: 
  - Tunneling service for easy internet access (no port forwarding needed)
  - Managed updates and backups
  - Priority support

### Future SaaS Offering
The architecture is designed for multi-tenancy (though not implemented yet). The plan:
- **Free tier**: Limited rooms and storage
- **Pro tier**: Unlimited rooms, advanced analytics, custom branding
- **Enterprise**: Persisted Logs, SSO integration, compliance features, dedicated infrastructure
- **...**

The beauty of this approach: customers can start self-hosted and migrate to SaaS later, or run hybrid deployments. No lock-in, just convenience.

## Documentation

- **[Backend](backend/README.md)**
- **[Iac](infrastructure/README.md)**
- **[Sequence Diagrams](docs/sequence-diagram/)**
- **[ERD](docs/data/erd.mermaid)**
- **[System Architecture](docs/architecture/simple-architecture.mermaid)**

## Future Improvements
- **Better video quality controls**: Manual quality selection, bandwidth adaptation settings
- **Multi-tenancy implementation**: Complete SaaS backend with proper tenant isolation
- **Latency optimization**: Video preloading, caching improvements, etc.
- **Security hardening**: Rate limiting, input validation, audit w immutable logs, etc.

---

*Built with the bootstrapper's philosophy: start simple, ship fast, scale smart.*