// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"
)

const (
	projectFinalizer = "project.app.terraform.io/finalizer"
)

func TestIsCreationCandidate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		project  Project
		expected bool
	}{
		"HasID": {
			Project{Status: ProjectStatus{ID: "prj-this"}},
			false,
		},
		"DoesNotHaveID": {
			Project{Status: ProjectStatus{ID: ""}},
			true,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			out := c.project.IsCreationCandidate()
			if out != c.expected {
				t.Fatalf("Error matching output and expected: %#v vs %#v", out, c.expected)
			}
		})
	}
}
