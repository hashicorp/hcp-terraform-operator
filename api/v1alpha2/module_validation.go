// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (m *Module) ValidateSpec() error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, m.validateSpecWorkspace()...)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: GroupVersion.Group, Kind: "Module"},
		m.Name,
		allErrs,
	)
}

func (m *Module) validateSpecWorkspace() field.ErrorList {
	allErrs := field.ErrorList{}

	if m.Spec.Workspace.ID == "" && m.Spec.Workspace.Name == "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec"),
			"workspace",
			"one of ID or Name must be set"),
		)
	}

	if m.Spec.Workspace.ID != "" && m.Spec.Workspace.Name != "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec"),
			"workspace",
			"only one of ID or Name can be used at a time, not both"),
		)
	}

	return allErrs
}
