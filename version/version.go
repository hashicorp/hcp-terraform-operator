// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package version

import "fmt"

var (
	// The version should be 'X.0.0-dev' during the whole development cycle of a particular major version X.
	// Minor and patch parts should remain unchanged.
	Version = "2.0.0-dev"
	Source  = fmt.Sprintf("HashiCorp/KubernetesOperator/v%s", Version)
)
