terraform {
  required_providers {
    digitalocean = { source = "registry.terraform.io/digitalocean/digitalocean", version = "2.100.0" }
    cloudflare   = { source = "registry.terraform.io/cloudflare/cloudflare", version = "5.24.0" }
    upstash      = { source = "registry.terraform.io/upstash/upstash", version = "2.1.0" }
    random       = { source = "registry.opentofu.org/hashicorp/random", version = "3.7.2" }
  }
}
