// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"
)

func TestValidateWorkspaceSpecAgentPool(t *testing.T) {
	successCases := map[string]Workspace{
		"HasOnlyID": {
			Spec: WorkspaceSpec{
				AgentPool: &WorkspaceAgentPool{
					ID: "this",
				},
			},
		},
		"HasOnlyName": {
			Spec: WorkspaceSpec{
				AgentPool: &WorkspaceAgentPool{
					Name: "this",
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecAgentPool(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Workspace{
		"HasIDandName": {
			Spec: WorkspaceSpec{
				AgentPool: &WorkspaceAgentPool{
					ID:   "this",
					Name: "this",
				},
			},
		},
		"HasEmptyIDandName": {
			Spec: WorkspaceSpec{
				AgentPool: &WorkspaceAgentPool{},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecAgentPool(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}

func TestValidateWorkspaceSpecRemoteStateSharing(t *testing.T) {
	successCases := map[string]Workspace{
		"HasOnlyAllWorkspaces": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					AllWorkspaces: true,
				},
			},
		},
		"HasOnlyWorkspacesWithID": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					Workspaces: []*ConsumerWorkspace{
						{
							ID: "this",
						},
						{
							ID: "self",
						},
					},
				},
			},
		},
		"HasOnlyWorkspacesWithName": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					Workspaces: []*ConsumerWorkspace{
						{
							Name: "this",
						},
						{
							Name: "self",
						},
					},
				},
			},
		},
		"HasOnlyWorkspacesWithIDandName": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					Workspaces: []*ConsumerWorkspace{
						{
							ID: "this",
						},
						{
							Name: "self",
						},
					},
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecRemoteStateSharing(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Workspace{
		"HasAllWorkspacesAndWorkspacesWithID": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					AllWorkspaces: true,
					Workspaces: []*ConsumerWorkspace{
						{
							ID: "this",
						},
						{
							ID: "self",
						},
					},
				},
			},
		},
		"HasAllWorkspacesAndWorkspacesWithName": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					AllWorkspaces: true,
					Workspaces: []*ConsumerWorkspace{
						{
							Name: "this",
						},
						{
							Name: "self",
						},
					},
				},
			},
		},
		"HasAllWorkspacesAndWorkspacesWithIDandName": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					AllWorkspaces: true,
					Workspaces: []*ConsumerWorkspace{
						{
							ID: "this",
						},
						{
							Name: "self",
						},
					},
				},
			},
		},
		"HasEmptyAllWorkspacesAndWorkspaces": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{},
			},
		},
		"HasDuplicateWorkspacesName": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					Workspaces: []*ConsumerWorkspace{
						{
							Name: "this",
						},
						{
							Name: "this",
						},
					},
				},
			},
		},
		"HasDuplicateWorkspacesID": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					Workspaces: []*ConsumerWorkspace{
						{
							ID: "this",
						},
						{
							ID: "this",
						},
					},
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecRemoteStateSharing(); len(errs) == 0 {
				// fmt.Println(errs)
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}

func TestValidateWorkspaceSpecRunTrigger(t *testing.T) {
	successCases := map[string]Workspace{
		"HasOnlyID": {
			Spec: WorkspaceSpec{
				RunTriggers: []RunTrigger{
					{
						ID: "this",
					},
				},
			},
		},
		"HasOnlyName": {
			Spec: WorkspaceSpec{
				RunTriggers: []RunTrigger{
					{
						Name: "this",
					},
				},
			},
		},
		"HasOneWithIDandOneWithName": {
			Spec: WorkspaceSpec{
				RunTriggers: []RunTrigger{
					{
						ID: "this",
					},
					{
						Name: "this",
					},
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecRunTrigger(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Workspace{
		"HasIDandName": {
			Spec: WorkspaceSpec{
				RunTriggers: []RunTrigger{
					{
						ID:   "this",
						Name: "this",
					},
				},
			},
		},
		"HasEmptyIDandName": {
			Spec: WorkspaceSpec{
				RunTriggers: []RunTrigger{
					{
						Name: "",
						ID:   "",
					},
				},
			},
		},
		"HasDuplicateID": {
			Spec: WorkspaceSpec{
				RunTriggers: []RunTrigger{
					{
						ID: "this",
					},
					{
						ID: "this",
					},
				},
			},
		},
		"HasDuplicateName": {
			Spec: WorkspaceSpec{
				RunTriggers: []RunTrigger{
					{
						Name: "this",
					},
					{
						Name: "this",
					},
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecRunTrigger(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}

func TestValidateWorkspaceSpecSSHKey(t *testing.T) {
	successCases := map[string]Workspace{
		"HasOnlyID": {
			Spec: WorkspaceSpec{
				SSHKey: &SSHKey{
					ID: "this",
				},
			},
		},
		"HasOnlyName": {
			Spec: WorkspaceSpec{
				SSHKey: &SSHKey{
					Name: "this",
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecSSHKey(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Workspace{
		"HasIDandName": {
			Spec: WorkspaceSpec{
				SSHKey: &SSHKey{
					ID:   "this",
					Name: "this",
				},
			},
		},
		"HasEmptyIDandName": {
			Spec: WorkspaceSpec{
				SSHKey: &SSHKey{},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecSSHKey(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}

// TODO:Validation
//
// + EnvironmentVariables names duplicate: spec.environmentVariables[].name
// + TerraformVariables names duplicate: spec.terraformVariables[].name
// + Tags duplicate: spec.tags[]
// + AgentPool must be set when ExecutionMode = 'agent': spec.agentPool <- spec.executionMode['agent']
