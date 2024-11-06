// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateSpec validates the VarSet fields.
func (vs *WorkspaceVarSet) ValidateSpec() field.ErrorList {
	var allErrs field.ErrorList

	// Validate VarSetID
	if err := validateSpecVarSetID(vs.VarSetID); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("varSetID"), vs.VarSetID, err.Error()))
	}

	// Validate VarSetName
	if err := validateSpecVarSetName(vs.VarSetName); err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("varSetName"), vs.VarSetName, err.Error()))
	}

	return allErrs
}

// validateSpecVarSetID checks if the VarSetID matches the required pattern.
func validateSpecVarSetID(id string) error {
	varSetIDPattern := `^varset-[a-zA-Z0-9]+$`
	matched, err := regexp.MatchString(varSetIDPattern, id)
	if err != nil {
		return fmt.Errorf("failed to validate varSetID: %v", err)
	}
	if !matched {
		return fmt.Errorf("varSetID must match the pattern: %s", varSetIDPattern)
	}
	return nil
}

func validateSpecVarSetName(varSetName string) error {
	// Check if varSetName meets the minimum length requirement.
	if len(varSetName) < 1 {
		return fmt.Errorf("varSetName must be at least 1 character long")
	}

	return nil
}
