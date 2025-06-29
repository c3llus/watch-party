resource "google_artifact_registry_repository" "docker_repo" {
  project       = var.gcp_project_id
  location      = var.gcp_region
  repository_id = "app-services"
  description   = "Docker repository for watch party services"
  format        = "DOCKER"
  depends_on    = [google_project_service.apis]
}

resource "google_storage_bucket" "videos" {
  project                     = var.gcp_project_id
  name                        = "${var.gcp_project_id}-videos"
  location                    = var.gcp_region
  force_destroy               = false
  uniform_bucket_level_access = true

  cors {
    origin          = ["https://localhost:5173", "http://localhost:5173", "http://localhost:3000"]
    method          = ["GET", "HEAD", "PUT", "POST", "DELETE"]
    response_header = ["Content-Type", "Authorization", "Range", "Accept-Ranges"]
    max_age_seconds = 3600
  }

  lifecycle_rule {
    condition {
      age = 365
    }
    action {
      type = "Delete"
    }
  }

  versioning {
    enabled = true
  }
}

resource "random_password" "db_password" {
  length           = 24
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

resource "google_secret_manager_secret" "db_password" {
  project   = var.gcp_project_id
  secret_id = "db-password"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_version" "db_password_v1" {
  secret      = google_secret_manager_secret.db_password.id
  secret_data = random_password.db_password.result

  lifecycle {
    ignore_changes = [secret_data]
  }
}

resource "google_secret_manager_secret" "jwt_secret" {
  project   = var.gcp_project_id
  secret_id = "jwt-secret"

  replication {
    auto {}
  }

  depends_on = [google_project_service.apis]
}

resource "random_password" "jwt_secret" {
  length  = 64
  special = true
}

resource "google_secret_manager_secret_version" "jwt_secret_v1" {
  secret      = google_secret_manager_secret.jwt_secret.id
  secret_data = random_password.jwt_secret.result

  lifecycle {
    ignore_changes = [secret_data]
  }
}
