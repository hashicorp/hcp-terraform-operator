// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"
)

func TestValidateVarSetSpec(t *testing.T) {
	t.Parallel()

	successCases := map[string]WorkspaceVarSet{
		"ValidVarSet": {
			VarSetID:   "varset-validID",
			VarSetName: "valid-name",
			Scope:      "global",
		},
		"ValidVarSetWithProjectScope": {
			VarSetID:   "varset-validID",
			VarSetName: "new-valid-name",
			Scope:      "project",
		},
	}

	for name, c := range successCases {
		t.Run(name, func(t *testing.T) {
			if errs := c.ValidateSpec(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]WorkspaceVarSet{
		"EmptyVarSetID": {
			VarSetID:   "",
			VarSetName: "valid-name",
			Scope:      "global",
		},
		"InvalidVarSetID": {
			VarSetID:   "invalidID",
			VarSetName: "valid-name",
			Scope:      "global",
		},
		"EmptyVarSetName": {
			VarSetID:   "varset-validID",
			VarSetName: "",
			Scope:      "global",
		},
		"InvalidScope": {
			VarSetID:   "varset-validID",
			VarSetName: "valid-name",
			Scope:      "invalid-scope",
		},
		"AllEmptyFields": {
			VarSetID:   "",
			VarSetName: "",
			Scope:      "",
		},
	}

	for name, c := range errorCases {
		t.Run(name, func(t *testing.T) {
			if errs := c.ValidateSpec(); len(errs) == 0 {
				t.Error("Expected validation error but got none")
			}
		})
	}
}
