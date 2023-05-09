# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

variable "tfe-api-url" {
  type    = string
  default = "https://tfe.gcp.terraform-k8s-providers-ci.hashicorp.services/"
}

variable "admin_username" {
  type    = string
  default = "admin"
}

variable "admin_password" {
  type = string
  sensitive = true
}

variable "admin_email" {
  type    = string
}

data "http" "wait-for-ok" {
  url = "${var.tfe-api-url}/_health_check"
  retry {
    attempts     = 2000
    min_delay_ms = 1000
  }
}

data "http" "iact_token" {
  url = "${var.tfe-api-url}/admin/retrieve-iact"
  retry {
    attempts     = 2000
    min_delay_ms = 1000
  }
}

data "http" "admin_user_token" {
  url    = "${var.tfe-api-url}/admin/initial-admin-user?token=${data.http.iact_token.response_body}"
  method = "POST"
  request_headers = {
    Content-Type = "application/json"
  }
  request_body = jsonencode({
    username = var.admin_username
    password = var.admin_password
    email    = var.admin_email
  })
}

output "admin_token" {
  value = jsondecode(data.http.admin_user_token.response_body)
}
