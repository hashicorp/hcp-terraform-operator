// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateModuleSpecWorkspace(t *testing.T) {
	t.Parallel()

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
			errs := c.validateSpecWorkspace()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
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
			errs := c.validateSpecWorkspace()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}
