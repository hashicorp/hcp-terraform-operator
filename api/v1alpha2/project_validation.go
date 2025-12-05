// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (p *Project) ValidateSpec() error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, p.validateSpecTeamAccess()...)

	if len(allErrs) == 0 {
		return nil
	}

	return kerrors.NewInvalid(
		schema.GroupKind{Group: "", Kind: "Project"},
		p.Name,
		allErrs,
	)
}

func (p *Project) validateSpecTeamAccess() field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, p.validateSpecTeamAccessCustom()...)
	allErrs = append(allErrs, p.validateSpecTeamAccessTeam()...)

	return allErrs
}

func (p *Project) validateSpecTeamAccessCustom() field.ErrorList {
	allErrs := field.ErrorList{}

	for i, ta := range p.Spec.TeamAccess {
		f := field.NewPath("spec").Child(fmt.Sprintf("[%d]", i))
		if ta.Access == tfc.TeamProjectAccessCustom {
			if ta.Custom == nil {
				allErrs = append(allErrs, field.Required(
					f,
					fmt.Sprintf("'spec.teamAccess.custom' must be set when 'spec.teamAccess' is set to %q", tfc.TeamProjectAccessCustom),
				))
			}
		} else {
			if ta.Custom != nil {
				allErrs = append(allErrs, field.Invalid(
					f,
					"",
					fmt.Sprintf("'spec.teamAccess.custom' cannot be used when 'spec.teamAccess' is set to %s", ta.Access),
				))
			}
		}
	}

	return allErrs
}

func (p *Project) validateSpecTeamAccessTeam() field.ErrorList {
	allErrs := field.ErrorList{}

	tai := make(map[string]int)
	tan := make(map[string]int)

	for i, ta := range p.Spec.TeamAccess {
		f := field.NewPath("spec").Child(fmt.Sprintf("[%d]", i))
		if ta.Team.ID == "" && ta.Team.Name == "" {
			allErrs = append(allErrs, field.Invalid(
				f,
				"",
				"one of the field ID or Name must be set"),
			)
		}

		if ta.Team.ID != "" && ta.Team.Name != "" {
			allErrs = append(allErrs, field.Invalid(
				f,
				"",
				"only one of the field ID or Name is allowed"),
			)
		}

		if ta.Team.ID != "" {
			if _, ok := tai[ta.Team.ID]; ok {
				allErrs = append(allErrs, field.Duplicate(f.Child("ID"), ta.Team.ID))
			}
			tai[ta.Team.ID] = i
		}

		if ta.Team.Name != "" {
			if _, ok := tan[ta.Team.Name]; ok {
				allErrs = append(allErrs, field.Duplicate(f.Child("Name"), ta.Team.Name))
			}
			tan[ta.Team.Name] = i
		}
	}

	return allErrs
}
