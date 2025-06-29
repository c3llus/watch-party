variable "gcp_project_id" {
  description = "The GCP Project ID."
  type        = string
}

variable "gcp_region" {
  description = "The GCP region to deploy resources in."
  type        = string
  default     = "asia-southeast1"
}

variable "api_image_url" {
  description = "The full URL of the API service Docker image."
  type        = string
}

variable "sync_image_url" {
  description = "The full URL of the Sync service Docker image."
  type        = string
}
