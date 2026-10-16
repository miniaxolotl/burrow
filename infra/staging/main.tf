variable "image_digest" { type = string }
variable "registry_credentials" {
  type      = string
  sensitive = true
  default   = ""
}
variable "domain_validation_records" {
  type    = map(object({ name = string, value = string }))
  default = {}
}

module "tunnels" {
  source                    = "../modules/tunnel-environment"
  environment               = "staging"
  domain                    = "staging.brmbl.dev"
  dns_zone_id               = "ed5bfd6f3ff1470abfc2ea6ee2dae99d"
  website_zone_id           = "a74b445d8aed01725d46a050a0662585"
  image_digest              = var.image_digest
  registry_credentials      = var.registry_credentials
  domain_validation_records = var.domain_validation_records
  website_records = {
    "staging.useburrow.app" = "512b6a6e860cfec4.vercel-dns-016.com"
  }
}

output "app_id" { value = module.tunnels.app_id }
output "app_name" { value = module.tunnels.app_name }
output "url" { value = module.tunnels.url }
output "image_digest" { value = module.tunnels.image_digest }
output "domain_validation_records" { value = module.tunnels.domain_validation_records }
