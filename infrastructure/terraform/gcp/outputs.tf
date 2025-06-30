output "api_service_url" {
  description = "The URL of the deployed API service"
  value       = google_cloud_run_v2_service.api.uri
}

output "sync_service_backend" {
  description = "The backend service for the WebSocket sync service (Global Load Balancer)"
  value       = google_compute_backend_service.sync_global_backend.id
}

output "sync_service_internal_ip" {
  description = "The global load balancer backend service for SSL termination"
  value       = "Global backend service: ${google_compute_backend_service.sync_global_backend.name}"
}

output "sync_service_instance_group" {
  description = "The instance group running the sync service"
  value       = google_compute_region_instance_group_manager.sync_igm.instance_group
}

output "database_connection_name" {
  description = "The connection name of the Cloud SQL instance"
  value       = google_sql_database_instance.postgres.connection_name
}

output "database_private_ip" {
  description = "The private IP address of the Cloud SQL instance"
  value       = google_sql_database_instance.postgres.private_ip_address
}

output "redis_host" {
  description = "The host IP of the Redis instance"
  value       = google_redis_instance.redis.host
}

output "redis_port" {
  description = "The port of the Redis instance"
  value       = google_redis_instance.redis.port
}

output "videos_bucket_name" {
  description = "The name of the videos storage bucket"
  value       = google_storage_bucket.videos.name
}

output "docker_repository_url" {
  description = "The URL of the Docker repository"
  value       = "${var.gcp_region}-docker.pkg.dev/${var.gcp_project_id}/${google_artifact_registry_repository.docker_repo.repository_id}"
}

output "vpc_network_name" {
  description = "The name of the VPC network"
  value       = google_compute_network.vpc_network.name
}

output "serverless_connector_name" {
  description = "The name of the VPC Access connector"
  value       = google_vpc_access_connector.serverless.name
}

output "websocket_vm_external_ip" {
  description = "External IP address of the WebSocket service VM"
  value       = length(google_compute_region_instance_group_manager.sync_igm.status) > 0 ? "Check GCP Console for instance IP" : "No instances running"
}

output "compute_service_account_email" {
  description = "Service account email for the compute instances (for Ansible authentication)"
  value       = google_service_account.compute_sa.email
}

output "db_host" {
  description = "Database host (private IP)"
  value       = google_sql_database_instance.postgres.private_ip_address
}

output "db_name" {
  description = "Database name"
  value       = google_sql_database.main_db.name
}

output "db_username" {
  description = "Database username"
  value       = google_sql_user.users.name
}

output "db_password_secret_id" {
  description = "Secret Manager secret ID for database password"
  value       = google_secret_manager_secret.db_password.secret_id
}

output "jwt_secret_secret_id" {
  description = "Secret Manager secret ID for JWT secret"
  value       = google_secret_manager_secret.jwt_secret.secret_id
}

output "load_balancer_ip" {
  description = "The global IP address of the load balancer (for WebSocket SSL connections)"
  value       = google_compute_global_address.websocket_lb_ip.address
}

output "websocket_endpoint" {
  description = "The HTTPS/WSS endpoint for WebSocket connections"
  value       = "wss://${google_compute_global_address.websocket_lb_ip.address}"
}
