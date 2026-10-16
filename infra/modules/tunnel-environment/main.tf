locals {
  name = var.environment == "production" ? "burrow" : "burrow-staging"
}

resource "digitalocean_project" "this" {
  name        = local.name
  description = "Burrow ${var.environment} tunnel service"
  purpose     = "Service or API"
  environment = title(var.environment)
}

resource "upstash_redis_database" "tunnels" {
  database_name  = local.name
  region         = "global"
  primary_region = "us-east-1"
  tls            = true
  eviction       = false
  auto_scale     = false
  budget         = 20 # Upstash's minimum monthly usage cap, not a fixed charge.
}

resource "digitalocean_app" "tunnels" {
  project_id = digitalocean_project.this.id
  spec {
    name                      = local.name
    region                    = "tor"
    disable_edge_cache        = true
    disable_email_obfuscation = true

    domain {
      name     = var.domain
      type     = "PRIMARY"
      wildcard = true
    }

    ingress {
      rule {
        component {
          name                 = "burrowd"
          preserve_path_prefix = true
        }
        match {
          path { prefix = "/" }
        }
      }
    }

    service {
      name               = "burrowd"
      instance_count     = 1 # WebSocket/yamux sessions are held in process memory.
      instance_size_slug = "apps-s-1vcpu-0.5gb"
      http_port          = 25701

      image {
        registry_type        = "GHCR"
        registry             = "miniaxolotl"
        repository           = "burrowd"
        digest               = var.image_digest
        registry_credentials = var.registry_credentials
      }

      env {
        key   = "BURROW_DOMAIN"
        value = var.domain
        scope = "RUN_TIME"
      }
      env {
        key   = "BURROW_TLS"
        value = "true"
        scope = "RUN_TIME"
      }
      env {
        key   = "BURROW_REDIS_URL"
        value = "rediss://default:${urlencode(upstash_redis_database.tunnels.password)}@${upstash_redis_database.tunnels.endpoint}:${upstash_redis_database.tunnels.port}"
        scope = "RUN_TIME"
        type  = "SECRET"
      }

      health_check {
        http_path             = "/health"
        initial_delay_seconds = 10
        period_seconds        = 10
        timeout_seconds       = 5
        success_threshold     = 1
        failure_threshold     = 3
      }
      termination {
        grace_period_seconds = 30
        drain_seconds        = 5
      }
    }
  }
}

resource "cloudflare_dns_record" "tunnels" {
  for_each = toset([var.domain, "*.${var.domain}"])
  zone_id  = var.dns_zone_id
  name     = each.value
  type     = "CNAME"
  content  = trimsuffix(trimprefix(digitalocean_app.tunnels.default_ingress, "https://"), "/")
  proxied  = false
  ttl      = 1
}

resource "cloudflare_dns_record" "validation" {
  for_each = var.domain_validation_records
  zone_id  = var.dns_zone_id
  name     = each.value.name
  type     = "TXT"
  content  = each.value.value
  ttl      = 60
}

resource "cloudflare_dns_record" "website" {
  for_each = var.website_records
  zone_id  = var.website_zone_id
  name     = each.key
  type     = "CNAME"
  content  = each.value
  proxied  = false
  ttl      = 1
}
