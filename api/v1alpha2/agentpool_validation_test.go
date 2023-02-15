// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"
)

func TestValidateAgentPoolSpecAgentToken(t *testing.T) {
	successCases := map[string]AgentPool{
		"HasOnlyName": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentToken{
					{
						Name: "this",
					},
				},
			},
		},
		"HasMultipleTokens": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentToken{
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
				AgentTokens: []*AgentToken{
					{
						Name: "this",
						ID:   "this",
					},
				},
			},
		},
		"HasCreatedAt": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentToken{
					{
						Name:      "this",
						CreatedAt: PointerOf(int64(1984)),
					},
				},
			},
		},
		"HasLastUsedAt": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentToken{
					{
						Name:       "this",
						LastUsedAt: PointerOf(int64(1984)),
					},
				},
			},
		},
		"HasDuplicateName": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentToken{
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
