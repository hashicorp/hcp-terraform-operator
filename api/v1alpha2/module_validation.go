// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (m *Module) ValidateSpec() error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, m.validateSpecWorkspace()...)

	if len(allErrs) == 0 {
		return nil
	}

	return kerrors.NewInvalid(
		schema.GroupKind{Group: "", Kind: "Module"},
		m.Name,
		allErrs,
	)
}

func (m *Module) validateSpecWorkspace() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := m.Spec.Workspace
	f := field.NewPath("spec").Child("workspace")

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

// TODO:Validation
//
// + Variables names duplicate: spec.variables[].name
// + Outputs names duplicate: spec.outputs[].name
//
// + Invalid CR cannot be deleted until it is fixed -- need to discuss if we want to do something about it
