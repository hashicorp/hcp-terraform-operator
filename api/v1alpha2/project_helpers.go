// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

func (p *Project) IsCreationCandidate() bool {
	return p.Status.ID == ""
}
