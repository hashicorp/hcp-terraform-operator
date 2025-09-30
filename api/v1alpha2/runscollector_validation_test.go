// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import "testing"

func TestValidateRunsCollectorSpecAgentPool(t *testing.T) {
	t.Parallel()

	successCases := map[string]RunsCollector{
		"HasOnlyID": {
			Spec: RunsCollectorSpec{
				AgentPool: &AgentPoolRef{
					ID: "this",
				},
			},
		},
		"HasOnlyName": {
			Spec: RunsCollectorSpec{
				AgentPool: &AgentPoolRef{
					Name: "this",
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecAgentPool(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]RunsCollector{
		"HasIDandName": {
			Spec: RunsCollectorSpec{
				AgentPool: &AgentPoolRef{
					ID:   "this",
					Name: "this",
				},
			},
		},
		"HasEmptyIDandName": {
			Spec: RunsCollectorSpec{
				AgentPool: &AgentPoolRef{},
			},
		},
		"HasInvalidExecutionMode": {
			Spec: RunsCollectorSpec{
				AgentPool: &AgentPoolRef{
					ID:   "this",
					Name: "this",
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecAgentPool(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}
