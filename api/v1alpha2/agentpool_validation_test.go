// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateAgentPoolSpecAgentToken(t *testing.T) {
	t.Parallel()

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
						CreatedAt: pointer.PointerOf(int64(1984)),
					},
				},
			},
		},
		"HasLastUsedAt": {
			Spec: AgentPoolSpec{
				AgentTokens: []*AgentToken{
					{
						Name:       "this",
						LastUsedAt: pointer.PointerOf(int64(1984)),
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

func TestValidateLabels(t *testing.T) {
	tests := []struct {
		name    string
		labels  map[string]string
		wantErr bool
	}{
		{
			name: "valid labels",
			labels: map[string]string{
				"key": "my-value",
			},
			wantErr: false,
		},
		{
			name: "empty key",
			labels: map[string]string{
				"": "my-value",
			},
			wantErr: true,
		},
		{
			name: "empty value",
			labels: map[string]string{
				"key": "",
			},
			wantErr: true,
		},
	}

	for _, l := range tests {
		t.Run(l.name, func(t *testing.T) {
			errs := validateLabels(l.labels, field.NewPath("spec").Child("labels"))
			if (len(errs) > 0) != l.wantErr {
				t.Errorf("validateLabels() error = %v, wantErr %v", errs, l.wantErr)
			}
		})
	}
}

func TestValidateAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		wantErr     bool
	}{
		{
			name: "valid annotations",
			annotations: map[string]string{
				"key": "my-value",
			},
			wantErr: false,
		},
		{
			name: "empty key",
			annotations: map[string]string{
				"": "my-value",
			},
			wantErr: true,
		},
		{
			name: "empty value",
			annotations: map[string]string{
				"key": "",
			},
			wantErr: true,
		},
	}

	for _, a := range tests {
		t.Run(a.name, func(t *testing.T) {
			errs := validateAnnotations(a.annotations, field.NewPath("spec").Child("annotations"))
			if (len(errs) > 0) != a.wantErr {
				t.Errorf("validateAnnotations() error = %v, wantErr %v", errs, a.wantErr)
			}
		})
	}
}
