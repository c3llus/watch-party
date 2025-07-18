resource "google_compute_network" "vpc_network" {
  project                 = var.gcp_project_id
  name                    = "app-vpc"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "vpc_subnet" {
  project       = var.gcp_project_id
  name          = "app-subnet"
  ip_cidr_range = "10.0.0.0/24"
  region        = var.gcp_region
  network       = google_compute_network.vpc_network.id
}

resource "google_compute_global_address" "redis_peering_range" {
  project       = var.gcp_project_id
  name          = "redis-peering-range"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.vpc_network.id
}

resource "google_service_networking_connection" "private_service_connection" {
  network                 = google_compute_network.vpc_network.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.redis_peering_range.name]
}

resource "google_vpc_access_connector" "serverless" {
  project       = var.gcp_project_id
  name          = "serverless-connector"
  region        = var.gcp_region
  network       = google_compute_network.vpc_network.id
  ip_cidr_range = "10.8.0.0/28"
  depends_on    = [google_project_service.apis]
}

resource "google_sql_database_instance" "postgres" {
  project             = var.gcp_project_id
  name                = "watch-party-db-instance"
  region              = var.gcp_region
  database_version    = "POSTGRES_17"
  deletion_protection = false

  settings {
    tier = "db-f1-micro"
    ip_configuration {
      ipv4_enabled    = false
      private_network = google_compute_network.vpc_network.id
    }
  }
  depends_on = [google_service_networking_connection.private_service_connection]
}

resource "google_sql_database" "main_db" {
  project  = var.gcp_project_id
  instance = google_sql_database_instance.postgres.name
  name     = "watch_party_db"
}

resource "google_sql_user" "users" {
  project  = var.gcp_project_id
  instance = google_sql_database_instance.postgres.name
  name     = "app-user"
  password = random_password.db_password.result
}

resource "google_redis_instance" "redis" {
  project            = var.gcp_project_id
  name               = "watch-party-redis-instance"
  tier               = "BASIC"
  memory_size_gb     = 1
  region             = var.gcp_region
  authorized_network = google_compute_network.vpc_network.id
  connect_mode       = "DIRECT_PEERING"
  depends_on         = [google_service_networking_connection.private_service_connection]
}

resource "google_cloud_run_v2_service" "api" {
  project  = var.gcp_project_id
  name     = "service-api"
  location = var.gcp_region
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    service_account = google_service_account.cloud_run_sa.email

    containers {
      image = var.api_image_url

      env {
        name  = "ENVIRONMENT"
        value = "production"
      }
      env {
        name  = "GCP_PROJECT_ID"
        value = var.gcp_project_id
      }
      env {
        name  = "GOOGLE_CLOUD_PROJECT"
        value = var.gcp_project_id
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
    }

    vpc_access {
      connector = google_vpc_access_connector.serverless.id
      egress    = "ALL_TRAFFIC"
    }

    scaling {
      min_instance_count = 0
      max_instance_count = 10
    }
  }

  depends_on = [google_project_service.apis]
}

resource "google_service_account" "compute_sa" {
  project      = var.gcp_project_id
  account_id   = "compute-sync-service"
  display_name = "Service Account for Compute Engine Sync Service"
}

resource "google_secret_manager_secret_iam_member" "compute_db_secret" {
  project   = var.gcp_project_id
  secret_id = google_secret_manager_secret.db_password.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.compute_sa.email}"
}

resource "google_secret_manager_secret_iam_member" "compute_jwt_secret" {
  project   = var.gcp_project_id
  secret_id = google_secret_manager_secret.jwt_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.compute_sa.email}"
}

resource "google_project_iam_member" "compute_sa_secret_manager_accessor" {
  project = var.gcp_project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.compute_sa.email}"
}

resource "google_project_iam_member" "compute_sa_artifact_registry_reader" {
  project = var.gcp_project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.compute_sa.email}"
}

data "google_compute_zones" "available" {
  project = var.gcp_project_id
  region  = var.gcp_region
}

resource "google_compute_instance_template" "sync_template" {
  project      = var.gcp_project_id
  name_prefix  = "service-sync-template-"
  machine_type = "e2-micro"
  region       = var.gcp_region

  disk {
    source_image = "cos-cloud/cos-stable"
    auto_delete  = true
    boot         = true
    disk_type    = "pd-standard"
    disk_size_gb = 10
  }

  network_interface {
    network    = google_compute_network.vpc_network.id
    subnetwork = google_compute_subnetwork.vpc_subnet.id
    access_config {}
  }

  service_account {
    email  = google_service_account.compute_sa.email
    scopes = ["cloud-platform"]
  }

  tags = ["websocket-service"]

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [google_project_service.apis]
}

resource "google_compute_region_instance_group_manager" "sync_igm" {
  project                   = var.gcp_project_id
  name                      = "service-sync-igm"
  base_instance_name        = "service-sync"
  region                    = var.gcp_region
  distribution_policy_zones = data.google_compute_zones.available.names

  version {
    instance_template = google_compute_instance_template.sync_template.id
  }

  target_size = 1

  auto_healing_policies {
    health_check      = google_compute_health_check.sync_health_check.id
    initial_delay_sec = 300
  }

  named_port {
    name = "websocket"
    port = 8080
  }

  depends_on = [google_compute_instance_template.sync_template]
}

resource "google_compute_health_check" "sync_health_check" {
  project             = var.gcp_project_id
  name                = "service-sync-health-check"
  check_interval_sec  = 10
  timeout_sec         = 5
  healthy_threshold   = 2
  unhealthy_threshold = 3

  http_health_check {
    port         = 8080
    request_path = "/health"
  }

  depends_on = [google_project_service.apis]
}

resource "google_compute_firewall" "allow_ssh" {
  project = var.gcp_project_id
  name    = "allow-ssh-access"
  network = google_compute_network.vpc_network.id

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["websocket-service"]

  depends_on = [google_project_service.apis]
}

resource "google_compute_firewall" "allow_websocket" {
  project = var.gcp_project_id
  name    = "allow-websocket-traffic"
  network = google_compute_network.vpc_network.id

  allow {
    protocol = "tcp"
    ports    = ["8080", "443"]
  }

  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["websocket-service"]

  depends_on = [google_project_service.apis]
}

# Regional backend service - disabled in favor of global backend service for SSL termination
# resource "google_compute_region_backend_service" "sync_backend" {
#   project  = var.gcp_project_id
#   name     = "service-sync-backend"
#   region   = var.gcp_region
#   protocol = "TCP"

#   backend {
#     group = google_compute_region_instance_group_manager.sync_igm.instance_group
#   }

#   health_checks = [google_compute_health_check.sync_health_check.id]

#   session_affinity = "CLIENT_IP"

#   depends_on = [
#     google_compute_region_instance_group_manager.sync_igm,
#     google_compute_health_check.sync_health_check
#   ]
# }

resource "google_cloud_run_service_iam_binding" "api_noauth" {
  project  = var.gcp_project_id
  location = google_cloud_run_v2_service.api.location
  service  = google_cloud_run_v2_service.api.name
  role     = "roles/run.invoker"
  members = [
    "allUsers",
    "serviceAccount:${google_service_account.cloud_run_sa.email}",
    "serviceAccount:${google_service_account.compute_sa.email}",
  ]
}

resource "google_service_account" "cloud_run_sa" {
  project      = var.gcp_project_id
  account_id   = "cloud-run-services"
  display_name = "Service Account for Cloud Run Services"
}

resource "google_secret_manager_secret_iam_member" "cloud_run_db_secret" {
  project   = var.gcp_project_id
  secret_id = google_secret_manager_secret.db_password.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_run_sa.email}"
}

resource "google_secret_manager_secret_iam_member" "cloud_run_jwt_secret" {
  project   = var.gcp_project_id
  secret_id = google_secret_manager_secret.jwt_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_run_sa.email}"
}

resource "google_storage_bucket_iam_member" "cloud_run_storage" {
  bucket = google_storage_bucket.videos.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.cloud_run_sa.email}"
}

resource "google_project_iam_member" "cloud_run_service_agent" {
  project = var.gcp_project_id
  role    = "roles/run.serviceAgent"
  member  = "serviceAccount:${google_service_account.cloud_run_sa.email}"
}

resource "google_service_account_iam_binding" "github_actions_service_account_user" {
  service_account_id = google_service_account.cloud_run_sa.name
  role               = "roles/iam.serviceAccountUser"

  members = [
    "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
  ]
}

resource "google_project_iam_member" "github_actions_compute_os_login" {
  project = var.gcp_project_id
  role    = "roles/compute.osLogin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_compute_instance_admin" {
  project = var.gcp_project_id
  role    = "roles/compute.instanceAdmin.v1"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_compute_viewer" {
  project = var.gcp_project_id
  role    = "roles/compute.viewer"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_iam_security_admin" {
  project = var.gcp_project_id
  role    = "roles/iam.securityAdmin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_service_account_admin" {
  project = var.gcp_project_id
  role    = "roles/iam.serviceAccountAdmin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_cloud_run_admin" {
  project = var.gcp_project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_storage_admin" {
  project = var.gcp_project_id
  role    = "roles/storage.admin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_secret_manager_admin" {
  project = var.gcp_project_id
  role    = "roles/secretmanager.admin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_artifactregistry_admin" {
  project = var.gcp_project_id
  role    = "roles/artifactregistry.admin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_artifactregistry_writer" {
  project = var.gcp_project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_cloudsql_admin" {
  project = var.gcp_project_id
  role    = "roles/cloudsql.admin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_compute_admin" {
  project = var.gcp_project_id
  role    = "roles/compute.admin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_compute_network_admin" {
  project = var.gcp_project_id
  role    = "roles/compute.networkAdmin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_redis_admin" {
  project = var.gcp_project_id
  role    = "roles/redis.admin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_servicenetworking_service_agent" {
  project = var.gcp_project_id
  role    = "roles/servicenetworking.serviceAgent"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_vpcaccess_admin" {
  project = var.gcp_project_id
  role    = "roles/vpcaccess.admin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_serviceusage_admin" {
  project = var.gcp_project_id
  role    = "roles/serviceusage.serviceUsageAdmin"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}

resource "google_project_iam_member" "github_actions_service_account_user_project" {
  project = var.gcp_project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:github-actions-deployer@${var.gcp_project_id}.iam.gserviceaccount.com"
}
