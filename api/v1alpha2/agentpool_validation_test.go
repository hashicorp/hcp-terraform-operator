// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func TestValidateAgentPoolSpecAgentToken(t *testing.T) {
	t.Parallel()

	successCases := map[string]AgentPool{
		"HasOnlyName": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentPoolToken{
					{
						Name: "this",
					},
				},
			},
		},
		"HasMultipleTokens": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentPoolToken{
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
			if errs := c.validateSpecAgentToken(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]AgentPool{
		"HasID": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentPoolToken{
					{
						Name: "this",
						ID:   "this",
					},
				},
			},
		},
		"HasCreatedAt": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentPoolToken{
					{
						Name:      "this",
						CreatedAt: pointer.PointerOf(int64(1984)),
					},
				},
			},
		},
		"HasLastUsedAt": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentPoolToken{
					{
						Name:       "this",
						LastUsedAt: pointer.PointerOf(int64(1984)),
					},
				},
			},
		},
		"HasDuplicateName": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentPoolToken{
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
			if errs := c.validateSpecAgentToken(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}
