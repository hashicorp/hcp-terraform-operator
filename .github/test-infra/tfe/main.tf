# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

# Random String for unique names
# ------------------------------
resource "random_pet" "main" {
  length = 1
}

# Store TFE License as secret
# ---------------------------
module "secrets" {
  source = "github.com/hashicorp/terraform-google-terraform-enterprise/fixtures/secrets"

  license = {
    id   = random_pet.main.id
    path = var.license_file
  }
}

# Gets the external IP address of the provisioner
# TFE requires this in order to accept admin creation API calls
# -------------------------------------------------------------
data "http" "icanhazip" {
   url = "http://icanhazip.com"
}

# Standalone, mounted disk
# ------------------------
module "tfe" {
  source = "github.com/hashicorp/terraform-google-terraform-enterprise"

  distribution                = "ubuntu"
  dns_zone_name               = var.dns_zone_name
  existing_service_account_id = var.existing_service_account_id
  namespace                   = random_pet.main.id
  node_count                  = 1
  fqdn                        = var.fqdn
  load_balancer               = "PUBLIC"
  ssl_certificate_name        = var.ssl_certificate_name
  tfe_license_secret_id       = module.secrets.license_secret
  vm_machine_type             = "n1-standard-4"
  iact_subnet_list = tolist(["${chomp(data.http.icanhazip.body)}/32"])
}
