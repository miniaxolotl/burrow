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
  environment               = "production"
  domain                    = "brmbl.dev"
  dns_zone_id               = "ed5bfd6f3ff1470abfc2ea6ee2dae99d"
  website_zone_id           = "a74b445d8aed01725d46a050a0662585"
  image_digest              = var.image_digest
  registry_credentials      = var.registry_credentials
  domain_validation_records = var.domain_validation_records
  website_records = {
    "useburrow.app"     = "512b6a6e860cfec4.vercel-dns-016.com"
    "www.useburrow.app" = "512b6a6e860cfec4.vercel-dns-016.com"
  }
}

import {
  to = module.tunnels.cloudflare_dns_record.website["useburrow.app"]
  id = "a74b445d8aed01725d46a050a0662585/2b901dd7d409d268a46230ac70b9352b"
}
import {
  to = module.tunnels.cloudflare_dns_record.website["www.useburrow.app"]
  id = "a74b445d8aed01725d46a050a0662585/0f6d1c844d1e7465341790a9ef14527f"
}

output "app_id" { value = module.tunnels.app_id }
output "app_name" { value = module.tunnels.app_name }
output "url" { value = module.tunnels.url }
output "image_digest" { value = module.tunnels.image_digest }
output "domain_validation_records" { value = module.tunnels.domain_validation_records }
