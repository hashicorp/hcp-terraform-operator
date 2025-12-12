// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
	"github.com/stretchr/testify/assert"
)

func TestValidateAgentPoolSpecAgentToken(t *testing.T) {
	t.Parallel()

	successCases := map[string]AgentPool{
		"HasOnlyName": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentAPIToken{
					{
						Name: "this",
					},
				},
			},
		},
		"HasMultipleTokens": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentAPIToken{
					{
						Name: "this",
					},
					{
						Name: "self",
					},
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecAgentToken()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	errorCases := map[string]AgentPool{
		"HasID": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentAPIToken{
					{
						Name: "this",
						ID:   "this",
					},
				},
			},
		},
		"HasCreatedAt": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentAPIToken{
					{
						Name:      "this",
						CreatedAt: pointer.PointerOf(int64(1984)),
					},
				},
			},
		},
		"HasLastUsedAt": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentAPIToken{
					{
						Name:       "this",
						LastUsedAt: pointer.PointerOf(int64(1984)),
					},
				},
			},
		},
		"HasDuplicateName": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentAPIToken{
					{
						Name: "this",
					},
					{
						Name: "this",
					},
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecAgentToken()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}
