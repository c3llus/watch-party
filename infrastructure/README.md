# WatchParty Infrastructure

Simple, containerized infrastructure designed for the bootstrapper mindset: **pragmatic enough to scale, simple enough to understand completely**.

## Architecture Philosophy

**Why containers?** Because infrastructure should be **moveable**. Today it's GCP, tomorrow it might be AWS, next year it could be your own servers. Containers make that transition painless.

**Why not Kubernetes immediately?** Because complexity has a cost. We start with managed services (Cloud Run) that give us container benefits without operational overhead. When we outgrow managed services, migrating to GKE is straightforward since everything's already containerized.

**Bootstrapping mindset**: Build the simplest thing that can scale. Separate concerns without over-engineering. Make infrastructure decisions that don't lock us into specific vendors or patterns.

## Directory Structure

```
infrastructure/
├── terraform/          # Infrastructure as Code
│   ├── gcp/            # Google Cloud Platform resources
│   └── cloudflare/     # DNS and CDN configuration
├── ansible/            # Configuration management and deployment
│   ├── playbooks/      # Deployment automation
│   ├── inventory/      # Environment definitions
│   └── templates/      # Service configuration templates
└── bootstrap/          # Initial setup and state management
```

## Infrastructure Components

### Terraform (IaC)
**Purpose**: Declarative infrastructure management. Create, modify, and version cloud resources.

**GCP Resources** (`terraform/gcp/`):
- **Cloud Run**: Managed container platform for both API and Sync services
- **Cloud SQL**: Managed PostgreSQL for persistent data
- **Redis**: Managed Redis for session state and caching
- **Secret Manager**: Encrypted credential storage (DB passwords, JWT secrets)
- **Cloud Storage**: Object storage for video files and thumbnails
- **Artifact Registry**: Container image storage and versioning
- **Load Balancer**: HTTPS termination and traffic routing
- **VPC**: Private networking with secure service communication

**Cloudflare Resources** (`terraform/cloudflare/`):
- **DNS management**: Domain routing and subdomain configuration  
- **CDN**: Global edge caching for static assets
- **SSL certificates**: Automated certificate management
- **DDoS protection**: Built-in attack mitigation

### Ansible (Configuration Management)
**Purpose**: Automated deployment and configuration. Bridge the gap between infrastructure and application deployment.

**Playbooks** (`ansible/playbooks/`):
- `setup.yml`: Initial environment configuration and secrets injection
- `deploy.yml`: Rolling deployment with health checks and rollback capability
- `verify.yml`: Post-deployment validation and smoke tests
- `rollback.yml`: Automated rollback to previous stable version

**Why Ansible over pure CI/CD?** 
- **Flexibility**: Works with any CI system or manual execution
- **Idempotency**: Safe to run multiple times
- **Debugging**: Step-through capability for deployment troubleshooting

### Bootstrap (`bootstrap/`)
**Purpose**: One-time setup for Terraform state management and initial credentials.

Creates the foundational pieces:
- GCS bucket for Terraform state storage
- Initial service accounts and IAM permissions
- Basic networking and security groups

## Network Topology

```mermaid
graph TB
    subgraph "Internet"
        U[Users]
        CDN[Cloudflare CDN]
    end
    
    subgraph "GCP Project"
        subgraph "Load Balancer"
            LB[Global Load Balancer<br/>HTTPS Termination]
        end
        
        subgraph "VPC Network (10.0.0.0/24)"
            subgraph "Cloud Run Services"
                API[API Service<br/>:8080]
                SYNC[Sync Service<br/>:8081]
            end
            
            subgraph "Managed Databases"
                PG[(Cloud SQL<br/>PostgreSQL)]
                RD[(Redis<br/>Session Store)]
            end
            
            subgraph "Storage"
                GCS[Cloud Storage<br/>Videos & Assets]
                AR[Artifact Registry<br/>Container Images]
            end
            
            subgraph "Security"
                SM[Secret Manager<br/>Credentials]
            end
        end
    end
    
    %% External connections
    U -->|HTTPS| CDN
    CDN -->|Cache Miss| LB
    U -->|Direct for API| LB
    
    %% Load balancer routing
    LB -->|/api/*| API
    LB -->|/ws| SYNC
    
    %% Service dependencies
    API -->|Read/Write| PG
    API -->|Upload/Download| GCS
    API -->|Get Secrets| SM
    
    SYNC -->|Session State| RD
    SYNC -->|Get Secrets| SM
    
    %% Deployment
    AR -.->|Container Images| API
    AR -.->|Container Images| SYNC
    
    %% Styling
    classDef storage fill:#e1f5fe
    classDef service fill:#f3e5f5
    classDef database fill:#fff3e0
    classDef security fill:#ffebee
    
    class GCS,AR storage
    class API,SYNC service  
    class PG,RD database
    class SM security
```

## Service Communication

### External Traffic Flow
1. **Users** → **Cloudflare CDN** (static assets, caching)
2. **Users** → **Load Balancer** (API calls, WebSocket connections)
3. **Load Balancer** → **Cloud Run Services** (based on path routing)

### Internal Service Communication
- **API Service** ↔ **Cloud SQL**: User data, room configuration, video metadata
- **Sync Service** ↔ **Redis**: Real-time session state, room participants
- **Both Services** ↔ **Secret Manager**: Database credentials, JWT secrets
- **API Service** ↔ **Cloud Storage**: Video upload/download, thumbnail generation

### Security Boundaries
- **Private VPC**: All services communicate within isolated network
- **IAM-based access**: Service accounts with minimal necessary permissions
- **Secret management**: No hardcoded credentials; everything via Secret Manager
- **HTTPS everywhere**: TLS termination at load balancer, encrypted internal traffic

## Deployment Strategy

### Container-First Approach
Every service is containerized from day one:
- **Consistent environments**: Development, staging, production use identical containers
- **Easy scaling**: Container orchestration handles resource allocation
- **Simple rollbacks**: Previous container versions remain available
- **Provider flexibility**: Containers run anywhere

### Managed Services Strategy
Start with fully managed options, migrate to self-managed when beneficial:
- **Cloud Run** → **GKE** (when we need more control over networking/scheduling)
- **Cloud SQL** → **Self-hosted PostgreSQL** (when we need custom extensions or cost optimization)
- **Managed Redis** → **Redis Cluster** (when we need multi-region replication)

### GitOps Workflow (for GCE)
1. **Code changes** trigger container builds via GitHub Actions
2. **Container images** pushed to Artifact Registry with semantic versioning
3. **Direct SSH** into single Compute Engine instance
4. **Simple blue-green deployment** with docker stop/pull/run commands 
5. **Health checks** to container
6. **Rollbacks** to previous Docker image if health check fails

**Remarks**: Due to time constraints and bootstrapping mindset, the current deployment process isn't the best appraoch to handle this.

## Google Cloud Services Usage

### Core Infrastructure
- **Cloud Run**: Serverless container platform, automatic scaling from 0 to 1000+ instances
- **Cloud Load Balancing**: Global HTTPS load balancer with automatic SSL certificate management
- **VPC**: Private networking with service-to-service security

### Data Layer
- **Cloud SQL (PostgreSQL)**: Managed database with automated backups, high availability
- **Redis (Memorystore)**: Managed Redis for session storage and caching
- **Cloud Storage**: Object storage for videos, thumbnails, and static assets

### Security & Secrets
- **Secret Manager**: Centralized secret storage with automatic rotation capability
- **IAM**: Fine-grained access control with service account authentication

### DevOps & Monitoring
- **Artifact Registry**: Container image storage with vulnerability scanning
- **Cloud Logging**: Centralized log aggregation with structured query capability
- **Cloud Monitoring**: Application and infrastructure metrics with alerting

### Cost Optimization Features
- **Cloud Run**: Pay-per-request pricing, scales to zero when unused
- **Preemptible instances**: Cost savings for non-critical batch processing

### Secrets Management

- **Secret Manager**: Sensitive data (database passwords, API keys, JWT secrets)

---

*Infrastructure philosophy: Start simple, scale pragmatically. Every decision should make the next step easier, not harder.*
