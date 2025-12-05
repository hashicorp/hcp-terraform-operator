// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	tfc "github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateWorkspaceSpecAgentPool(t *testing.T) {
	t.Parallel()

	successCases := map[string]Workspace{
		"HasOnlyID": {
			Spec: WorkspaceSpec{
				AgentPool: &AgentPoolRef{
					ID: "this",
				},
				ExecutionMode: "agent",
			},
		},
		"HasOnlyName": {
			Spec: WorkspaceSpec{
				AgentPool: &AgentPoolRef{
					Name: "this",
				},
				ExecutionMode: "agent",
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecAgentPool()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	errorCases := map[string]Workspace{
		"HasIDandName": {
			Spec: WorkspaceSpec{
				AgentPool: &AgentPoolRef{
					ID:   "this",
					Name: "this",
				},
				ExecutionMode: "agent",
			},
		},
		"HasEmptyIDandName": {
			Spec: WorkspaceSpec{
				AgentPool:     &AgentPoolRef{},
				ExecutionMode: "agent",
			},
		},
		"HasInvalidExecutionMode": {
			Spec: WorkspaceSpec{
				AgentPool: &AgentPoolRef{
					ID:   "this",
					Name: "this",
				},
				ExecutionMode: "remote",
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecAgentPool()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateWorkspaceSpecExecutionMode(t *testing.T) {
	t.Parallel()

	successCases := map[string]Workspace{
		"AgentWithAgentPoolWithID": {
			Spec: WorkspaceSpec{
				ExecutionMode: "agent",
				AgentPool: &AgentPoolRef{
					ID: "this",
				},
			},
		},
		"AgentWithAgentPoolWithName": {
			Spec: WorkspaceSpec{
				ExecutionMode: "agent",
				AgentPool: &AgentPoolRef{
					Name: "this",
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecExecutionMode()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	errorCases := map[string]Workspace{
		"AgentWithoutAgentPool": {
			Spec: WorkspaceSpec{
				ExecutionMode: "agent",
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecExecutionMode()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateWorkspaceSpecNotifications(t *testing.T) {
	t.Parallel()

	token := "token"
	url := "https://www.hashicorp.com"
	successCases := map[string]Workspace{
		"OnlyEmailAddresses": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: tfc.NotificationDestinationTypeEmail,
						EmailAddresses: []string{
							"user@mail.com",
						},
					},
				},
			},
		},
		"OnlyEmailUsers": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: tfc.NotificationDestinationTypeEmail,
						EmailUsers: []string{
							"user@mail.com",
						},
					},
				},
			},
		},
		"EmailAddressesAndEmailUsers": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: tfc.NotificationDestinationTypeEmail,
						EmailAddresses: []string{
							"user@mail.com",
						},
						EmailUsers: []string{
							"user@mail.com",
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
						Type:  tfc.NotificationDestinationTypeGeneric,
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
						Type: tfc.NotificationDestinationTypeMicrosoftTeams,
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
						Type: tfc.NotificationDestinationTypeSlack,
						URL:  url,
					},
				},
			},
		},
		"AllTypes": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "thisA",
						Type: tfc.NotificationDestinationTypeEmail,
						EmailAddresses: []string{
							"user@mail.com",
						},
					},
					{
						Name: "thisB",
						Type: tfc.NotificationDestinationTypeEmail,
						EmailUsers: []string{
							"user@mail.com",
						},
					},
					{
						Name: "thisC",
						Type: tfc.NotificationDestinationTypeEmail,
						EmailAddresses: []string{
							"user@mail.com",
						},
						EmailUsers: []string{
							"user@mail.com",
						},
					},
					{
						Name:  "thisD",
						Type:  tfc.NotificationDestinationTypeGeneric,
						Token: token,
						URL:   url,
					},
					{
						Name: "thisE",
						Type: tfc.NotificationDestinationTypeMicrosoftTeams,
						URL:  url,
					},
					{
						Name: "thisF",
						Type: tfc.NotificationDestinationTypeSlack,
						URL:  url,
					},
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecNotifications()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	errorCases := map[string]Workspace{
		"EmailWithoutEmails": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: tfc.NotificationDestinationTypeEmail,
					},
				},
			},
		},
		"EmailWithUrl": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: tfc.NotificationDestinationTypeEmail,
						URL:  url,
					},
				},
			},
		},
		"EmailWithToken": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  tfc.NotificationDestinationTypeEmail,
						Token: token,
					},
				},
			},
		},
		"GenericWithoutToken": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name: "this",
						Type: tfc.NotificationDestinationTypeGeneric,
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
						Type:  tfc.NotificationDestinationTypeGeneric,
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
						Type:  tfc.NotificationDestinationTypeGeneric,
						Token: token,
						URL:   url,
						EmailAddresses: []string{
							"user@mail.com",
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
						Type:  tfc.NotificationDestinationTypeGeneric,
						Token: token,
						URL:   url,
						EmailUsers: []string{
							"user@mail.com",
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
						Type:  tfc.NotificationDestinationTypeMicrosoftTeams,
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
						Type: tfc.NotificationDestinationTypeMicrosoftTeams,
					},
				},
			},
		},
		"MicrosoftTeamsWithEmailAddresses": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  tfc.NotificationDestinationTypeMicrosoftTeams,
						Token: token,
						URL:   url,
						EmailAddresses: []string{
							"user@mail.com",
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
						Type:  tfc.NotificationDestinationTypeMicrosoftTeams,
						Token: token,
						URL:   url,
						EmailUsers: []string{
							"user@mail.com",
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
						Type:  tfc.NotificationDestinationTypeMicrosoftTeams,
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
						Type: tfc.NotificationDestinationTypeSlack,
					},
				},
			},
		},
		"SlackWithEmailAddresses": {
			Spec: WorkspaceSpec{
				Notifications: []Notification{
					{
						Name:  "this",
						Type:  tfc.NotificationDestinationTypeMicrosoftTeams,
						Token: token,
						URL:   url,
						EmailAddresses: []string{
							"user@mail.com",
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
						Type:  tfc.NotificationDestinationTypeMicrosoftTeams,
						Token: token,
						URL:   url,
						EmailUsers: []string{
							"user@mail.com",
						},
					},
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecNotifications()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateWorkspaceSpecRemoteStateSharing(t *testing.T) {
	t.Parallel()

	successCases := map[string]Workspace{
		"HasOnlyAllWorkspacesTrue": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					AllWorkspaces: true,
				},
			},
		},
		"HasOnlyAllWorkspacesFalse": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					AllWorkspaces: false,
				},
			},
		},
		"HasBothAllWorkspacesFalseAndWorkspacesWithName": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					AllWorkspaces: false,
					Workspaces: []*ConsumerWorkspace{
						{
							Name: "this",
						},
					},
				},
			},
		},
		"HasBothAllWorkspacesFalseAndWorkspacesWithID": {
			Spec: WorkspaceSpec{
				RemoteStateSharing: &RemoteStateSharing{
					AllWorkspaces: false,
					Workspaces: []*ConsumerWorkspace{
						{
							ID: "this",
						},
					},
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
			errs := c.validateSpecRemoteStateSharing()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
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
			errs := c.validateSpecRemoteStateSharing()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateWorkspaceSpecRunTasks(t *testing.T) {
	t.Parallel()

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
			errs := c.validateSpecRunTasks()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
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
			errs := c.validateSpecRunTasks()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateWorkspaceSpecRunTriggers(t *testing.T) {
	t.Parallel()

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
			errs := c.validateSpecRunTriggers()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
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
			errs := c.validateSpecRunTriggers()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateWorkspaceSpecSSHKey(t *testing.T) {
	t.Parallel()

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
			errs := c.validateSpecSSHKey()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
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
			errs := c.validateSpecSSHKey()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateWorkspaceSpecProject(t *testing.T) {
	t.Parallel()

	successCases := map[string]Workspace{
		"HasOnlyID": {
			Spec: WorkspaceSpec{
				Project: &WorkspaceProject{
					ID: "this",
				},
			},
		},
		"HasOnlyName": {
			Spec: WorkspaceSpec{
				Project: &WorkspaceProject{
					Name: "this",
				},
			},
		},
		"HasEmptyProject": {
			Spec: WorkspaceSpec{
				Project: nil,
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecProject()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	errorCases := map[string]Workspace{
		"HasIDandName": {
			Spec: WorkspaceSpec{
				Project: &WorkspaceProject{
					ID:   "this",
					Name: "this",
				},
			},
		},
		"HasEmptyIDandName": {
			Spec: WorkspaceSpec{
				Project: &WorkspaceProject{},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecProject()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateWorkspaceSpecVariables(t *testing.T) {
	t.Parallel()

	f := field.NewPath("spec")

	successCases := map[string][]Variable{
		"HasOnlyValue": {{
			Name:  "name",
			Value: "42",
		}},
		"HasOnlyValueFromConfigMapKeyRef": {{
			Name: "name",
			ValueFrom: &ValueFrom{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					Key:                  "this",
					LocalObjectReference: corev1.LocalObjectReference{Name: "this"},
				},
			},
		}},
		"HasOnlyValueFromSecretKeySelector": {{
			Name: "name",
			ValueFrom: &ValueFrom{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  "this",
					LocalObjectReference: corev1.LocalObjectReference{Name: "this"},
				},
			},
		}},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := validateSpecVariables(f.Child("terraformVariables"), c)
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)

			errs = validateSpecVariables(f.Child("environmentVariables"), c)
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	errorCases := map[string][]Variable{
		"HasDuplicateVariablesName": {
			{
				Name:  "name",
				Value: "42",
			},
			{
				Name:  "name",
				Value: "42",
			},
		},
		"HasValueAndValueFrom": {{
			Name:  "name",
			Value: "42",
			ValueFrom: &ValueFrom{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					Key:                  "this",
					LocalObjectReference: corev1.LocalObjectReference{Name: "this"},
				},
			},
		}},
		"HasEmptyValueFrom": {{
			Name:      "name",
			ValueFrom: &ValueFrom{},
		}},
		"HasValueFromConfigMapAndSecret": {{
			Name: "name",
			ValueFrom: &ValueFrom{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					Key:                  "this",
					LocalObjectReference: corev1.LocalObjectReference{Name: "this"},
				},
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  "this",
					LocalObjectReference: corev1.LocalObjectReference{Name: "this"},
				},
			},
		}},
		"HasValueFromConfigMapWithoutName": {{
			Name: "name",
			ValueFrom: &ValueFrom{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					Key: "this",
				},
			},
		}},
		"HasValueFromConfigMapWithoutKey": {{
			Name: "name",
			ValueFrom: &ValueFrom{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "this"},
				},
			},
		}},
		"HasValueFromSecretWithoutName": {{
			Name: "name",
			ValueFrom: &ValueFrom{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key: "this",
				},
			},
		}},
		"HasValueFromSecretWithoutKey": {{
			Name: "name",
			ValueFrom: &ValueFrom{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "this"},
				},
			},
		}},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := validateSpecVariables(f.Child("terraformVariables"), c)
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")

			errs = validateSpecVariables(f.Child("environmentVariables"), c)
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateSpecDeletionPolicy(t *testing.T) {
	t.Parallel()

	successCases := map[string]Workspace{
		"DeletionPolicyDestroyAllowDestroyPlanTrue": {
			Spec: WorkspaceSpec{
				AllowDestroyPlan: true,
				DeletionPolicy:   DeletionPolicyDestroy,
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			if errs := c.validateSpecDeletionPolicy(); len(errs) != 0 {
				assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
			}
		})
	}

	errorCases := map[string]Workspace{
		"DeletionPolicyDestroyAllowDestroyPlanFalse": {
			Spec: WorkspaceSpec{
				AllowDestroyPlan: false,
				DeletionPolicy:   DeletionPolicyDestroy,
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecDeletionPolicy()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateSpecVariableSets(t *testing.T) {
	t.Parallel()

	successCases := map[string]Workspace{
		"HasIDandName": {
			Spec: WorkspaceSpec{
				VariableSets: []WorkspaceVariableSet{
					{
						Name: "this", // Only Name is set
					},
				},
			},
		},
	}

	// Run Success Test Cases
	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecVariableSets()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	// Error Cases: Invalid combinations of ID and Name
	errorCases := map[string]Workspace{
		"HasEmptyIDandEmptyName": {
			Spec: WorkspaceSpec{
				VariableSets: []WorkspaceVariableSet{
					{
						ID:   "",
						Name: "", // Neither ID nor Name is set (invalid)
					},
				},
			},
		},

		"HasBothIDandName": {
			Spec: WorkspaceSpec{
				VariableSets: []WorkspaceVariableSet{
					{
						ID:   "thisID", // Both ID and Name are set (invalid)
						Name: "thisName",
					},
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecVariableSets()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}

func TestValidateSpecVersionControl(t *testing.T) {
	t.Parallel()

	successCases := map[string]Workspace{
		"HasNoVersionControlConfigured": {
			Spec: WorkspaceSpec{
				VersionControl: nil,
			},
		},
		"HasOnlyTriggerPatterns": {
			Spec: WorkspaceSpec{
				VersionControl: &VersionControl{
					EnableFileTriggers: true,
					TriggerPatterns:    []string{"path/*/workspace/*"},
				},
			},
		},
		"HasOnlyTriggerPrefixes": {
			Spec: WorkspaceSpec{
				WorkingDirectory: "path/",
				VersionControl: &VersionControl{
					EnableFileTriggers: true,
					TriggerPrefixes:    []string{"path/to/workspace/"},
				},
			},
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecVersionControl()
			assert.Empty(t, errs, "Unexpected validation errors: %v", errs)
		})
	}

	errorCases := map[string]Workspace{
		"TriggerPrefixesWithoutWorkingDirectory": {
			Spec: WorkspaceSpec{
				VersionControl: &VersionControl{
					EnableFileTriggers: true,
					TriggerPrefixes:    []string{"path/to/workspace/"},
				},
			},
		},
		"BothTriggerOptions": {
			Spec: WorkspaceSpec{
				WorkingDirectory: "path/",
				VersionControl: &VersionControl{
					EnableFileTriggers: true,
					TriggerPatterns:    []string{"path/*/workspace/*"},
					TriggerPrefixes:    []string{"path/to/workspace/"},
				},
			},
		},
		"TriggerPatternsWithoutEnableFileTriggers": {
			Spec: WorkspaceSpec{
				VersionControl: &VersionControl{
					TriggerPatterns: []string{"path/*/workspace/*"},
				},
			},
		},
		"TriggerPrefixesWithoutEnableFileTriggers": {
			Spec: WorkspaceSpec{
				VersionControl: &VersionControl{
					TriggerPrefixes: []string{"path/to/workspace/"},
				},
			},
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			errs := c.validateSpecVersionControl()
			assert.NotEmpty(t, errs, "Unexpected failure, at least one error is expected")
		})
	}
}
