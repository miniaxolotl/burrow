output "app_id" { value = digitalocean_app.tunnels.id }
output "app_name" { value = local.name }
output "url" { value = "https://${var.domain}" }
output "image_digest" { value = var.image_digest }
output "default_ingress" { value = digitalocean_app.tunnels.default_ingress }
output "domain_validation_records" { value = var.domain_validation_records }
output "redis_endpoint" { value = upstash_redis_database.tunnels.endpoint }
