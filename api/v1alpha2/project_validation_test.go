// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	tfc "github.com/hashicorp/go-tfe"
)

func TestValidateSpecTeamAccess(t *testing.T) {
	t.Parallel()

	successCases := map[string]Project{
		"CustomTeamAccessCustomsIsSet": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Access: tfc.TeamProjectAccessCustom,
						Custom: &CustomProjectPermissions{
							CreateWorkspace: true,
						},
					},
				},
			},
		},
		"AdminTeamAccessCustomIsNotSet": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Access: tfc.TeamProjectAccessAdmin,
						Custom: nil,
					},
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecTeamAccessCustom(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Project{
		"CustomTeamAccessCustomIsNotSet": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Access: tfc.TeamProjectAccessCustom,
						Custom: nil,
					},
				},
			},
		},
		"AdminTeamAccessCustomIsSet": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Access: tfc.TeamProjectAccessAdmin,
						Custom: &CustomProjectPermissions{
							CreateWorkspace: true,
						},
					},
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecTeamAccessCustom(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}

func TestValidateSpecTeamAccessTeam(t *testing.T) {
	t.Parallel()

	successCases := map[string]Project{
		"HasTeamsWithID": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Team: Team{
							ID: "this",
						},
					},
					{
						Team: Team{
							ID: "self",
						},
					},
				},
			},
		},
		"HasTeamsWithName": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Team: Team{
							Name: "this",
						},
					},
					{
						Team: Team{
							Name: "self",
						},
					},
				},
			},
		},
		"HasTeamsWithIDandName": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Team: Team{
							ID: "this",
						},
					},
					{
						Team: Team{
							Name: "self",
						},
					},
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecTeamAccessTeam(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Project{
		"HasTeamsWithDuplicateID": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Team: Team{
							ID: "this",
						},
					},
					{
						Team: Team{
							ID: "this",
						},
					},
				},
			},
		},
		"HasTeamsWithDuplicateName": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Team: Team{
							Name: "this",
						},
					},
					{
						Team: Team{
							Name: "this",
						},
					},
				},
			},
		},
		"HasTeamWithIDandName": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Team: Team{
							ID:   "this",
							Name: "this",
						},
					},
				},
			},
		},
		"HasTeamWithoutIDandName": {
			Spec: ProjectSpec{
				TeamAccess: []*ProjectTeamAccess{
					{
						Team: Team{},
					},
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecTeamAccessTeam(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}
