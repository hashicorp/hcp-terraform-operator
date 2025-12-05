// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (rc *RunsCollector) ValidateSpec() error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, rc.validateSpecAgentPool()...)

	if len(allErrs) == 0 {
		return nil
	}

	return kerrors.NewInvalid(
		schema.GroupKind{Group: "", Kind: "RunsCollector"},
		rc.Name,
		allErrs,
	)
}

func (rc *RunsCollector) validateSpecAgentPool() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := rc.Spec.AgentPool

	if spec == nil {
		return allErrs
	}

	f := field.NewPath("spec").Child("agentPool")

	if spec.ID == "" && spec.Name == "" {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			"one of the field ID or Name must be set"),
		)
	}

	if spec.ID != "" && spec.Name != "" {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			"only one of the field ID or Name is allowed"),
		)
	}

	return allErrs
}
