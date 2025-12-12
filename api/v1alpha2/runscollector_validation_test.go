// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			errs := c.validateSpecAgentPool()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
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
			errs := c.validateSpecAgentPool()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}
