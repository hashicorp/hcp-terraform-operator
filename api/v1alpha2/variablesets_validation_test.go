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
		},
		"ValidVarSetWithDifferentName": {
			VarSetID:   "varset-validID",
			VarSetName: "new-valid-name",
		},
	}

	for name, c := range successCases {
		t.Run(name, func(t *testing.T) {
			if errs := c.ValidateSpec(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors in %s: %v", name, errs)
			}
		})
	}

	errorCases := map[string]WorkspaceVarSet{
		"EmptyVarSetID": {
			VarSetID:   "",
			VarSetName: "valid-name",
		},
		"InvalidVarSetID": {
			VarSetID:   "invalidID",
			VarSetName: "valid-name",
		},
		"EmptyVarSetName": {
			VarSetID:   "varset-validID",
			VarSetName: "",
		},
		"AllEmptyFields": {
			VarSetID:   "",
			VarSetName: "",
		},
	}

	for name, c := range errorCases {
		t.Run(name, func(t *testing.T) {
			errs := c.ValidateSpec()
			if len(errs) == 0 {
				t.Error("Expected validation error but got none")
			} else {
				t.Logf("Validation errors in %s: %v", name, errs)
			}
		})
	}
}
