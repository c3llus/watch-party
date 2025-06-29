output "api_service_url" {
  description = "The URL of the deployed API service"
  value       = google_cloud_run_v2_service.api.uri
}

output "sync_service_backend" {
  description = "The backend service for the WebSocket sync service (Compute Engine)"
  value       = google_compute_region_backend_service.sync_backend.id
}

output "sync_service_internal_ip" {
  description = "The internal IP that can be used to access the sync service within the VPC"
  value       = "Regional backend service: ${google_compute_region_backend_service.sync_backend.name} in ${var.gcp_region}"
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
