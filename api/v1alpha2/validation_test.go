// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateLabels(t *testing.T) {
	tests := []struct {
		name    string
		labels  map[string]string
		wantErr bool
	}{
		{
			name: "HasValidLabel",
			labels: map[string]string{
				"key": "my-value",
			},
			wantErr: false,
		},
		{
			name: "HasEmptyKey",
			labels: map[string]string{
				"": "my-value",
			},
			wantErr: true,
		},
		{
			name: "HasEmptyValue",
			labels: map[string]string{
				"key": "",
			},
			wantErr: true,
		},
	}

	for _, l := range tests {
		t.Run(l.name, func(t *testing.T) {
			errs := validateLabels(l.labels, field.NewPath("metadata").Child("labels"))
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
			name: "HasValidAnnotations",
			annotations: map[string]string{
				"key": "my-value",
			},
			wantErr: false,
		},
		{
			name: "HasEmptyKey",
			annotations: map[string]string{
				"": "my-value",
			},
			wantErr: true,
		},
		{
			name: "HasEmptyValue",
			annotations: map[string]string{
				"key": "",
			},
			wantErr: true,
		},
	}

	for _, a := range tests {
		t.Run(a.name, func(t *testing.T) {
			errs := validateAnnotations(a.annotations, field.NewPath("metadata").Child("annotations"))
			if (len(errs) > 0) != a.wantErr {
				t.Errorf("validateAnnotations() error = %v, wantErr %v", errs, a.wantErr)
			}
		})
	}
}
