// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"
)

func TestValidateModuleSpecWorkspace(t *testing.T) {
	successCases := map[string]Module{
		"HasOnlyID": {
			Spec: ModuleSpec{
				Workspace: &ModuleWorkspace{
					ID: "this",
				},
			},
		},
		"HasOnlyName": {
			Spec: ModuleSpec{
				Workspace: &ModuleWorkspace{
					Name: "this",
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecWorkspace(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Module{
		"HasIDandName": {
			Spec: ModuleSpec{
				Workspace: &ModuleWorkspace{
					ID:   "this",
					Name: "this",
				},
			},
		},
		"HasEmptyIDandName": {
			Spec: ModuleSpec{
				Workspace: &ModuleWorkspace{},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecWorkspace(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}
