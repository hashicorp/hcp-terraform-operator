// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateDeploymentLabels(t *testing.T) {
	successCases := map[string]struct {
		labels map[string]string
	}{
		"HasValidLabel": {
			labels: map[string]string{
				"key": "my-value",
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := validateDeploymentLabels(c.labels, field.NewPath("metadata").Child("labels"))
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	errorCases := map[string]struct {
		labels map[string]string
	}{
		"HasEmptyKey": {
			labels: map[string]string{
				"": "my-value",
			},
		},
		"HasEmptyValue": {
			labels: map[string]string{
				"key": "",
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := validateDeploymentLabels(c.labels, field.NewPath("metadata").Child("labels"))
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateDeploymentAnnotations(t *testing.T) {
	successCases := map[string]struct {
		annotations map[string]string
	}{
		"HasValidAnnotations": {
			annotations: map[string]string{
				"key": "my-value",
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := validateDeploymentAnnotations(c.annotations, field.NewPath("metadata").Child("annotations"))
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	errorCases := map[string]struct {
		annotations map[string]string
	}{
		"HasEmptyKeys": {
			annotations: map[string]string{
				"": "my-value",
			},
		},
		"HasEmptyValue": {
			annotations: map[string]string{
				"key": "",
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := validateDeploymentAnnotations(c.annotations, field.NewPath("metadata").Child("annotations"))
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}
