// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (s *Stack) ValidateSpec() error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, s.validateSpecProject()...)
	allErrs = append(allErrs, s.validateSpecVCSRepo()...)
	allErrs = append(allErrs, s.validateSpecTerraformVariables()...)
	allErrs = append(allErrs, s.validateSpecEnvironmentVariables()...)
	allErrs = append(allErrs, s.validateSpecDeployment()...)

	if len(allErrs) == 0 {
		return nil
	}

	return kerrors.NewInvalid(
		schema.GroupKind{Group: "", Kind: "Stack"},
		s.Name,
		allErrs,
	)
}

func (s *Stack) validateSpecProject() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := s.Spec.Project

	if spec == nil {
		return allErrs
	}

	f := field.NewPath("spec").Child("project")

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

func (s *Stack) validateSpecVCSRepo() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := s.Spec.VCSRepo

	if spec == nil {
		return allErrs
	}

	f := field.NewPath("spec").Child("vcsRepo")

	if spec.Identifier == "" {
		allErrs = append(allErrs, field.Required(
			f.Child("identifier"),
			"identifier must be set when vcsRepo is specified"),
		)
	}

	return allErrs
}

func (s *Stack) validateSpecTerraformVariables() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := s.Spec.TerraformVariables

	if spec == nil {
		return allErrs
	}

	return validateSpecVariables(field.NewPath("spec").Child("terraformVariables"), spec)
}

func (s *Stack) validateSpecEnvironmentVariables() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := s.Spec.EnvironmentVariables

	if spec == nil {
		return allErrs
	}

	return validateSpecVariables(field.NewPath("spec").Child("environmentVariables"), spec)
}

func (s *Stack) validateSpecDeployment() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := s.Spec.Deployment

	if spec == nil {
		return allErrs
	}

	f := field.NewPath("spec").Child("deployment")

	if len(spec.Names) == 0 {
		allErrs = append(allErrs, field.Required(
			f.Child("names"),
			"at least one deployment name must be specified"),
		)
	}

	// Check for duplicate deployment names
	seen := make(map[string]bool)
	for i, name := range spec.Names {
		if name == "" {
			allErrs = append(allErrs, field.Invalid(
				f.Child("names").Index(i),
				name,
				"deployment name cannot be empty"),
			)
		}
		if seen[name] {
			allErrs = append(allErrs, field.Duplicate(
				f.Child("names").Index(i),
				name),
			)
		}
		seen[name] = true
	}

	return allErrs
}

// TODO:Validation
//
// + Invalid CR cannot be deleted until it is fixed -- need to discuss if we want to do something about it

// Made with Bob
