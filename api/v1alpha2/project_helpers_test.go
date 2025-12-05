// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			assert.Equalf(t, c.expected, out, "Error matching output and expected: %#v vs %#v", out, c.expected)
		})
	}
}
