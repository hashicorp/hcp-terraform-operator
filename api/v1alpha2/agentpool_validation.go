// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (ap *AgentPool) ValidateSpec() error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, ap.validateSpecAgentToken()...)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "", Kind: "AgentPool"},
		ap.Name,
		allErrs,
	)
}

func (ap *AgentPool) validateSpecAgentToken() field.ErrorList {
	allErrs := field.ErrorList{}
	atn := make(map[string]int)

	for i, at := range ap.Spec.AgentTokens {
		f := field.NewPath("spec").Child(fmt.Sprintf("agentTokens[%d]", i))

		if at.ID != "" {
			allErrs = append(allErrs, field.Forbidden(
				f.Child("id"),
				"id is not allowed in the spec"),
			)
		}
		if at.CreatedAt != nil {
			allErrs = append(allErrs, field.Forbidden(
				f.Child("createdAt"),
				"createdAt is not allowed in the spec"),
			)
		}
		if at.LastUsedAt != nil {
			allErrs = append(allErrs, field.Forbidden(
				f.Child("lastUsedAt"),
				"lastUsedAt is not allowed in the spec"),
			)
		}

		if _, ok := atn[at.Name]; ok {
			allErrs = append(allErrs, field.Duplicate(f.Child("name"), at.Name))
		}
		atn[at.Name] = i
	}

	return allErrs
}

// Validating labels to ensure key value pairs are not empty
func ValidateLabels(labels map[string]string, fldPath *field.Path) field.ErrorList {
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

// Validate annotations to ensure key value pairs are not empty
func ValidateAnnotations(annotations map[string]string, fldPath *field.Path) field.ErrorList {
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

// TODO:Validation
//
// + Invalid CR cannot be deleted until it is fixed -- need to discuss if we want to do something about it
