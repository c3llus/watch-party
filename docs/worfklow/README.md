# WatchParty Deployment Documentation

Five GitHub workflows handle our entire deployment pipeline. Each workflow has a specific job and can run independently.

## Workflows Overview

| Workflow | What it does | When it runs |
|----------|-------------|--------------|
| `infrastructure.yml` | Creates/updates GCP resources via Terraform | Manual trigger only |
| `deploy-backend.yml` | Deploys backend services | Push to main or manual |
| `deploy-frontend.yml` | Deploys React app to Cloudflare | Frontend changes |
| `manual-deploy.yml` | Emergency deployments | Manual only |
| `manual-release.yml` | Builds standalone executables | Manual only |

## Why This Architecture

We run two backend services: API service on Cloud Run (serverless) and WebSocket service on a VM (needs persistent connections). Frontend goes to Cloudflare Pages for global CDN.

## How Each Workflow Works

### Infrastructure Management (`infrastructure.yml`)

Runs Terraform to create or update GCP resources. Always requires manual approval for safety.

What it manages:
- Cloud Run services
- Compute Engine instances  
- Cloud SQL database
- Redis instance
- Load balancers and networking
- IAM permissions


### Backend Deployment (`deploy-backend.yml`)

Triggered automatically when you push backend code to main. Builds new Docker images and deploys them.

Process:
1. Build API and WebSocket images tagged with git commit SHA
2. Push images to Artifact Registry
3. Deploy API service to Cloud Run (happens fast)
4. Deploy WebSocket service to VM using blue-green strategy

Blue-green deployment means both old and new versions run briefly. If health checks pass, traffic switches to new version. If they fail, old version keeps running.

*can be improved further down the road with canary-like deployments.*

### Frontend Deployment (`deploy-frontend.yml`)

Deploys React app to Cloudflare Pages when frontend code changes.

Build process:
1. `npm run build` with production environment variables
2. Upload to Cloudflare Pages
3. Global CDN distribution happens automatically

### Emergency Deployment (`manual-deploy.yml`)

For when you need to deploy outside the normal process.

Use cases:
- Hotfix deployment
- Test specific image versions
- Rollback to previous version

Can force rebuild or use existing images from registry.

### Release Builder (`manual-release.yml`)

Creates standalone executables for all platforms. Embeds frontend, backend services, database, and storage into single binaries.

Builds for:
- Linux x64
- Windows x64  
- macOS Intel
- macOS Apple Silicon
