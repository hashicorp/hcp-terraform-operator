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

func TestValidateWorkspaceSpecNotifications(t *testing.T) {
	token := "token"
	url := "https://example.com"
	successCases := map[string]Workspace{
		"OnlyEmail": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: "email",
						EmailAddresses: []string{
							"linus@torvalds.fi",
						},
						EmailUsers: []string{
							"linus@torvalds.fi",
						},
					},
				},
			},
		},
		"OnlyGeneric": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "generic",
						Token: token,
						URL:   url,
					},
				},
			},
		},
		"OnlyMicrosoftTeams": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: "microsoft-teams",
						URL:  url,
					},
				},
			},
		},
		"OnlySlack": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: "slack",
						URL:  url,
					},
				},
			},
		},
		"AllTypes": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: "email",
						EmailAddresses: []string{
							"linus@torvalds.fi",
						},
						EmailUsers: []string{
							"linus@torvalds.fi",
						},
					},
					{
						Name:  "this",
						Type:  "generic",
						Token: token,
						URL:   url,
					},
					{
						Name: "this",
						Type: "microsoft-teams",
						URL:  url,
					},
					{
						Name: "this",
						Type: "slack",
						URL:  url,
					},
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecNotifications(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Workspace{
		"GenericWithoutToken": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: "generic",
						URL:  url,
					},
				},
			},
		},
		"GenericWithoutURL": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "generic",
						Token: token,
					},
				},
			},
		},
		"GenericWithEmailAddresses": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "generic",
						Token: token,
						URL:   url,
						EmailAddresses: []string{
							"linus@torvalds.fi",
						},
					},
				},
			},
		},
		"GenericWithEmailUsers": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "generic",
						Token: token,
						URL:   url,
						EmailUsers: []string{
							"linus@torvalds.fi",
						},
					},
				},
			},
		},
		"MicrosoftTeamsWithToken": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "microsoft-teams",
						URL:   url,
						Token: token,
					},
				},
			},
		},
		"MicrosoftTeamsWithoutURL": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: "microsoft-teams",
					},
				},
			},
		},
		"MicrosoftTeamsWithEmailAddresses": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "microsoft-teams",
						Token: token,
						URL:   url,
						EmailAddresses: []string{
							"linus@torvalds.fi",
						},
					},
				},
			},
		},
		"MicrosoftTeamsWithEmailUsers": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "microsoft-teams",
						Token: token,
						URL:   url,
						EmailUsers: []string{
							"linus@torvalds.fi",
						},
					},
				},
			},
		},
		"SlackWithToken": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "microsoft-teams",
						URL:   url,
						Token: token,
					},
				},
			},
		},
		"SlackWithoutURL": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: "slack",
					},
				},
			},
		},
		"SlackWithEmailAddresses": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "microsoft-teams",
						Token: token,
						URL:   url,
						EmailAddresses: []string{
							"linus@torvalds.fi",
						},
					},
				},
			},
		},
		"SlackWithEmailUsers": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  "microsoft-teams",
						Token: token,
						URL:   url,
						EmailUsers: []string{
							"linus@torvalds.fi",
						},
					},
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecNotifications(); len(errs) == 0 {
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

func TestValidateWorkspaceSpecRunTasks(t *testing.T) {
	successCases := map[string]Workspace{
		"HasOnlyID": {
			Spec: WorkspaceSpec{
				RunTasks: []WorkspaceRunTask{
					{
						ID: "this",
					},
				},
			},
		},
		"HasOnlyName": {
			Spec: WorkspaceSpec{
				RunTasks: []WorkspaceRunTask{
					{
						Name: "this",
					},
				},
			},
		},
		"HasOneWithIDandOneWithName": {
			Spec: WorkspaceSpec{
				RunTasks: []WorkspaceRunTask{
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
			if errs := c.validateSpecRunTasks(); len(errs) != 0 {
				t.Errorf("Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Workspace{
		"HasIDandName": {
			Spec: WorkspaceSpec{
				RunTasks: []WorkspaceRunTask{
					{
						ID:   "this",
						Name: "this",
					},
				},
			},
		},
		"HasEmptyIDandName": {
			Spec: WorkspaceSpec{
				RunTasks: []WorkspaceRunTask{
					{
						Name: "",
						ID:   "",
					},
				},
			},
		},
		"HasDuplicateID": {
			Spec: WorkspaceSpec{
				RunTasks: []WorkspaceRunTask{
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
				RunTasks: []WorkspaceRunTask{
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
			if errs := c.validateSpecRunTasks(); len(errs) == 0 {
				t.Error("Unexpected failure, at least one error is expected")
			}
		})
	}
}

func TestValidateWorkspaceSpecRunTriggers(t *testing.T) {
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
			if errs := c.validateSpecRunTriggers(); len(errs) != 0 {
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
			if errs := c.validateSpecRunTriggers(); len(errs) == 0 {
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
