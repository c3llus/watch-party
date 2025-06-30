# Add Google Cloud Load Balancer with SSL termination
# This will handle HTTPS/WSS termination and forward to your GCE instance

resource "google_compute_global_address" "websocket_lb_ip" {
  project = var.gcp_project_id
  name    = "websocket-lb-ip"
}

resource "google_compute_managed_ssl_certificate" "websocket_cert" {
  project = var.gcp_project_id
  name    = "websocket-ssl-cert"

  managed {
    domains = ["${google_compute_global_address.websocket_lb_ip.address}.nip.io"]
  }
}

# Create a global backend service for the global load balancer
resource "google_compute_backend_service" "sync_global_backend" {
  project                         = var.gcp_project_id
  name                           = "sync-global-backend"
  port_name                      = "http"
  protocol                       = "HTTP"
  timeout_sec                    = 30
  connection_draining_timeout_sec = 30

  backend {
    group = google_compute_region_instance_group_manager.sync_igm.instance_group
  }

  health_checks = [google_compute_health_check.sync_health_check.id]
}

resource "google_compute_url_map" "websocket_urlmap" {
  project         = var.gcp_project_id
  name            = "websocket-urlmap"
  default_service = google_compute_backend_service.sync_global_backend.id
}

resource "google_compute_target_https_proxy" "websocket_https_proxy" {
  project          = var.gcp_project_id
  name             = "websocket-https-proxy"
  url_map          = google_compute_url_map.websocket_urlmap.id
  ssl_certificates = [google_compute_managed_ssl_certificate.websocket_cert.id]
}

resource "google_compute_global_forwarding_rule" "websocket_forwarding_rule" {
  project               = var.gcp_project_id
  name                  = "websocket-forwarding-rule"
  target                = google_compute_target_https_proxy.websocket_https_proxy.id
  port_range            = "443"
  ip_address            = google_compute_global_address.websocket_lb_ip.address
  load_balancing_scheme = "EXTERNAL"
}

# Output the Load Balancer IP and domain
output "websocket_lb_ip" {
  description = "Load Balancer IP for WebSocket service"
  value       = google_compute_global_address.websocket_lb_ip.address
}

output "websocket_domain" {
  description = "Domain for WebSocket service with SSL"
  value       = "${google_compute_global_address.websocket_lb_ip.address}.nip.io"
}
