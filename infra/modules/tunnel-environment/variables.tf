variable "environment" {
  type = string
  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "Use staging or production."
  }
}

variable "domain" { type = string }
variable "dns_zone_id" { type = string }

variable "image_digest" {
  type = string
  validation {
    condition     = can(regex("^sha256:[0-9a-f]{64}$", var.image_digest))
    error_message = "Deploy an immutable GHCR sha256 image digest."
  }
}

variable "registry_credentials" {
  type      = string
  sensitive = true
  default   = ""
}

variable "domain_validation_records" {
  description = "App Platform wildcard certificate TXT records, discovered by the deployment script."
  type        = map(object({ name = string, value = string }))
  default     = {}
}

variable "website_records" {
  type    = map(string)
  default = {}
}
variable "website_zone_id" { type = string }
