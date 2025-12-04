// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNeedUpdateStatus(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		rc       RunsCollector
		expected bool
	}{
		"NilStatus": {
			RunsCollector{
				Status: RunsCollectorStatus{AgentPool: nil},
			},
			true,
		},
		"SpecNameDoesNotMatchStatusName": {
			RunsCollector{
				Spec: RunsCollectorSpec{
					AgentPool: &AgentPoolRef{
						Name: "this",
					},
				},
				Status: RunsCollectorStatus{
					AgentPool: &AgentPoolRef{
						Name: "that",
					},
				},
			},
			true,
		},
		"SpecIDDoesNotMatchStatusID": {
			RunsCollector{
				Spec: RunsCollectorSpec{
					AgentPool: &AgentPoolRef{
						ID: "this",
					},
				},
				Status: RunsCollectorStatus{
					AgentPool: &AgentPoolRef{
						ID: "that",
					},
				},
			},
			true,
		},
		"SpecNameMatchesStatusName": {
			RunsCollector{
				Spec: RunsCollectorSpec{
					AgentPool: &AgentPoolRef{
						Name: "this",
					},
				},
				Status: RunsCollectorStatus{
					AgentPool: &AgentPoolRef{
						Name: "this",
					},
				},
			},
			false,
		},
		"SpecIDMatchesStatusID": {
			RunsCollector{
				Spec: RunsCollectorSpec{
					AgentPool: &AgentPoolRef{
						ID: "this",
					},
				},
				Status: RunsCollectorStatus{
					AgentPool: &AgentPoolRef{
						ID: "this",
					},
				},
			},
			false,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			out := c.rc.NeedUpdateStatus()
			assert.Equalf(t, c.expected, out, "Error matching output and expected: %#v vs %#v", out, c.expected)
		})
	}
}
