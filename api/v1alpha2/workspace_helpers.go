// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"github.com/hashicorp/hcp-terraform-operator/internal/slice"
)

func (w *Workspace) IsCreationCandidate() bool {
	return w.Status.WorkspaceID == ""
}

// AddOrUpdateVariableStatus adds a given variable to the status if it does not exist there; otherwise, it updates it.
func (s *WorkspaceStatus) AddOrUpdateVariableStatus(variable VariableStatus) {
	for i, v := range s.Variables {
		if v.Name == variable.Name && v.Category == variable.Category {
			s.Variables[i].ID = variable.ID
			s.Variables[i].VersionID = variable.VersionID
			s.Variables[i].ValueID = variable.ValueID
			s.Variables[i].Category = variable.Category
			return
		}
	}

	s.Variables = append(s.Variables, VariableStatus{
		Name:      variable.Name,
		ID:        variable.ID,
		VersionID: variable.VersionID,
		ValueID:   variable.ValueID,
		Category:  variable.Category,
	})
}

// GetVariableStatus returns a given variable from the status if it exists there; otherwise, nil.
func (s *WorkspaceStatus) GetVariableStatus(variable VariableStatus) *VariableStatus {
	for _, v := range s.Variables {
		if v.Name == variable.Name && v.Category == variable.Category {
			return &v
		}
	}

	return nil
}

// DeleteVariableStatus deletes a given variable from the status.
func (s *WorkspaceStatus) DeleteVariableStatus(variable VariableStatus) {
	for i, v := range s.Variables {
		if v.Name == variable.Name && v.Category == variable.Category {
			s.Variables = slice.RemoveFromSlice(s.Variables, i)
			return
		}
	}
}
