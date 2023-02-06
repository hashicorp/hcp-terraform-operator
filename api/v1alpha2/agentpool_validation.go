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
		schema.GroupKind{Group: GroupVersion.Group, Kind: "AgentPool"},
		ap.Name,
		allErrs,
	)
}

func (ap *AgentPool) validateSpecAgentToken() field.ErrorList {
	allErrs := field.ErrorList{}

	for i, at := range ap.Spec.AgentTokens {
		if at.ID != "" {
			allErrs = append(allErrs, field.Forbidden(
				field.NewPath("spec").Child("agentTokens").Child(fmt.Sprint(i)),
				"id is not allowed in the spec"),
			)
		}
		if at.CreatedAt != nil {
			allErrs = append(allErrs, field.Forbidden(
				field.NewPath("spec").Child("agentTokens").Child(fmt.Sprint(i)),
				"createdAt is not allowed in the spec"),
			)
		}
		if at.LastUsedAt != nil {
			allErrs = append(allErrs, field.Forbidden(
				field.NewPath("spec").Child("agentTokens").Child(fmt.Sprint(i)),
				"lastUsedAt is not allowed in the spec"),
			)
		}
	}

	return allErrs
}
