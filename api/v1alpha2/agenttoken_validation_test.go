// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func TestValidateAgentTokenSpecAgentToken(t *testing.T) {
	t.Parallel()

	successCases := map[string]AgentToken{
		"HasOnlyName": {
			Spec: AgentTokenSpec{
				AgentTokens: []AgentAPIToken{
					{
						Name: "this",
					},
				},
			},
		},
		"HasMultipleTokens": {
			Spec: AgentTokenSpec{
				AgentTokens: []AgentAPIToken{
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
			if errs := c.validateSpecAgentTokens(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]AgentToken{
		"HasID": {
			Spec: AgentTokenSpec{
				AgentTokens: []AgentAPIToken{
					{
						Name: "this",
						ID:   "this",
					},
				},
			},
		},
		"HasCreatedAt": {
			Spec: AgentTokenSpec{
				AgentTokens: []AgentAPIToken{
					{
						Name:      "this",
						CreatedAt: pointer.PointerOf(int64(1984)),
					},
				},
			},
		},
		"HasLastUsedAt": {
			Spec: AgentTokenSpec{
				AgentTokens: []AgentAPIToken{
					{
						Name:       "this",
						LastUsedAt: pointer.PointerOf(int64(1984)),
					},
				},
			},
		},
		"HasDuplicateName": {
			Spec: AgentTokenSpec{
				AgentTokens: []AgentAPIToken{
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
			if errs := c.validateSpecAgentTokens(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}
