// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAgentPoolIsCreationCandidate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		agentPool AgentPool
		expected  bool
	}{
		"HasID": {
			AgentPool{Status: AgentPoolStatus{AgentPoolID: "apool-this"}},
			false,
		},
		"DoesNotHaveID": {
			AgentPool{Status: AgentPoolStatus{AgentPoolID: ""}},
			true,
		},
		// A CR bound to a pre-existing pool must never be a creation candidate, even with empty
		// status -- otherwise losing status.agentPoolID (e.g. a cluster rebuild) would make the
		// controller create a duplicate pool and orphan the real one.
		"DoesNotHaveIDButReferencesExistingPool": {
			AgentPool{Spec: AgentPoolSpec{ExistingAgentPoolID: "apool-existing"}},
			false,
		},
		"HasIDAndReferencesExistingPool": {
			AgentPool{
				Spec:   AgentPoolSpec{ExistingAgentPoolID: "apool-existing"},
				Status: AgentPoolStatus{AgentPoolID: "apool-this"},
			},
			false,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			out := c.agentPool.IsCreationCandidate()
			assert.Equalf(t, c.expected, out, "Error matching output and expected: %#v vs %#v", out, c.expected)
		})
	}
}

func TestAgentPoolIsExistingPoolReference(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		agentPool AgentPool
		expected  bool
	}{
		"ReferencesExistingPool": {
			AgentPool{Spec: AgentPoolSpec{ExistingAgentPoolID: "apool-existing"}},
			true,
		},
		"DoesNotReferenceExistingPool": {
			AgentPool{Spec: AgentPoolSpec{ExistingAgentPoolID: ""}},
			false,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			out := c.agentPool.IsExistingPoolReference()
			assert.Equalf(t, c.expected, out, "Error matching output and expected: %#v vs %#v", out, c.expected)
		})
	}
}
