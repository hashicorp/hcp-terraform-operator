// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

func (w *Workspace) IsCreationCandidate() bool {
	return w.Status.WorkspaceID == ""
}
