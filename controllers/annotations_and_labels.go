// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

const (
	agentPoolAnnotationTokenRefreshDuration   = "agentpool.app.terraform.io/token-refresh-duration"
	agentPoolAnnotationTokenRetentionDuration = "agentpool.app.terraform.io/token-retention-duration"

	workspaceAnnotationRunNew              = "workspace.app.terraform.io/run-new"
	workspaceAnnotationRunType             = "workspace.app.terraform.io/run-type"
	workspaceAnnotationRunTerraformVersion = "workspace.app.terraform.io/run-terraform-version"
)
