// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Validating Deployment labels to ensure key value pairs are not empty
func validateDeploymentLabels(labels map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for k, v := range labels {
		if k == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("key"), "key must not be empty"))
		}
		if v == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("value"), "value must not be empty"))
		}
	}
	return allErrs
}

// Validate Deployment annotations to ensure key value pairs are not empty
func validateDeploymentAnnotations(annotations map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	for k, v := range annotations {
		if k == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("key"), "key must not be empty"))
		}
		if v == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("value"), "value must not be empty"))
		}
	}
	return allErrs
}

// TODO:
// - Add annotation validation for all controllers.
//   For example, 'app.terraform.io/paused' should only be set to 'true' or 'false'.
