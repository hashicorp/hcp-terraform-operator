# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "admin_email" {
  type    = string
  default = "tf-strategic@hashicorp.com"
}

data "terraform_remote_state" "tfe" {
  backend = "local"

  config = {
    path = "../terraform.tfstate"
  }
}

# Wait for the TFE installation to finish internal provisioning on the node and report itself as available.
#
data "http" "wait-for-ok" {
  url = data.terraform_remote_state.tfe.outputs.health_check_url
  retry {
    attempts     = 2000
    min_delay_ms = 1000
  }
}

# Grab the time-limited IACT token required to create the admin user.
#
data "http" "iact_token" {
  url = data.terraform_remote_state.tfe.outputs.iact_url
  retry {
    attempts     = 2000
    min_delay_ms = 1000
  }
}

# Create the admin user and retrieve its associated token
#
data "http" "admin_user_token" {
  url    = "${data.terraform_remote_state.tfe.outputs.initial_admin_user_url}?token=${data.http.iact_token.response_body}"
  method = "POST"
  request_headers = {
    Content-Type = "application/json"
  }
  request_body = jsonencode({
    username = "admin"
    password = data.terraform_remote_state.tfe.outputs.replicated_console_password
    email    = var.admin_email
  })
}

output "console_url" {
  value = data.terraform_remote_state.tfe.outputs.url
}

output "admin_password" {
  value = data.terraform_remote_state.tfe.outputs.replicated_console_password
}

output "admin_token" {
  value = jsondecode(data.http.admin_user_token.response_body).token
}
