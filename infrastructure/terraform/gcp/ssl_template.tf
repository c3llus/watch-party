# Add self-signed SSL certificate generation to your instance template
resource "google_compute_instance_template" "sync_template_ssl" {
  project      = var.gcp_project_id
  name_prefix  = "service-sync-ssl-template-"
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

  metadata_startup_script = <<-EOF
    #!/bin/bash
    
    # Create SSL directory
    mkdir -p /etc/ssl/private /etc/ssl/certs
    
    # Generate self-signed certificate for the instance IP
    INSTANCE_IP=$(curl -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip)
    
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
      -keyout /etc/ssl/private/server.key \
      -out /etc/ssl/certs/server.crt \
      -subj "/C=US/ST=CA/L=SF/O=WatchParty/CN=$INSTANCE_IP" \
      -addext "subjectAltName=IP:$INSTANCE_IP"
    
    # Start your container with SSL support
    docker run -d \
      --name watch-party-sync-ssl \
      --restart unless-stopped \
      -p 443:8443 \
      -p 80:8080 \
      -v /etc/ssl:/etc/ssl:ro \
      -e ENVIRONMENT=production \
      -e SSL_ENABLED=true \
      -e SSL_CERT_PATH=/etc/ssl/certs/server.crt \
      -e SSL_KEY_PATH=/etc/ssl/private/server.key \
      asia-southeast1-docker.pkg.dev/nafas-id-marcellus/app-services/service-sync:latest
  EOF

  lifecycle {
    create_before_destroy = true
  }

  depends_on = [google_project_service.apis]
}
