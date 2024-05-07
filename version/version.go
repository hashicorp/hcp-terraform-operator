// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package version

import "fmt"

var (
	// The version should remain as 'X.0.0-dev' throughout the entire development cycle of a specific major version X.
	// The minor and patch components should remain unchanged.
	Version   = "2.0.0-dev"
	UserAgent = fmt.Sprintf("HCPTerraformOperator/v%s", Version)
	// The user agent 'TerraformCloudOperator' was only used for version 2.3.0 and will remain here for visibility.
	// It is not commented out to ensure that future generations will not miss it.
	_ = fmt.Sprintf("TerraformCloudOperator/v%s", Version)
)
