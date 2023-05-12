# Copyright (c) HashiCorp, Inc.
# SPDX-License-Identifier: MPL-2.0

output "health_check_url" {
  value       = module.tfe.health_check_url
  description = "The URL of the Terraform Enterprise health check endpoint."
}

output "iact_notice" {
  value       = "Once deployed, please follow this page to set the initial user up: https://www.terraform.io/docs/enterprise/install/automating-initial-user.html"
  description = "Login advice message."
}

output "iact_url" {
  value       = module.tfe.iact_url
  description = "IACT URL"
}

output "initial_admin_user_url" {
  value       = module.tfe.initial_admin_user_url
  description = "Initial Admin user URL"
}

output "lb_address" {
  value       = module.tfe.lb_address
  description = "Load Balancer Address"
}

output "replicated_console_password" {
  value       = module.tfe.replicated_console_password
  description = "Generated password for replicated dashboard"
}

output "url" {
  value       = module.tfe.url
  description = "Login URL to setup the TFE instance once it is initialized"
}