// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (w *Workspace) ValidateSpec() error {
	var allErrs field.ErrorList

	allErrs = append(allErrs, w.validateSpecAgentPool()...)
	allErrs = append(allErrs, w.validateSpecNotifications()...)
	allErrs = append(allErrs, w.validateSpecRemoteStateSharing()...)
	allErrs = append(allErrs, w.validateSpecRunTasks()...)
	allErrs = append(allErrs, w.validateSpecRunTriggers()...)
	allErrs = append(allErrs, w.validateSpecSSHKey()...)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "", Kind: "Workspace"},
		w.Name,
		allErrs,
	)
}

func (w *Workspace) validateSpecAgentPool() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := w.Spec.AgentPool

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

func (w *Workspace) validateSpecNotifications() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := w.Spec.Notifications

	if spec == nil {
		return allErrs
	}

	nn := make(map[string]int)

	for i, n := range spec {
		f := field.NewPath("spec").Child("notifications").Child(fmt.Sprintf("[%d]", i))
		if _, ok := nn[n.Name]; ok {
			allErrs = append(allErrs, field.Duplicate(f.Child("Name"), n.Name))
		}
		nn[n.Name] = i
		switch n.Type {
		case "email":
			allErrs = append(allErrs, w.validateSpecNotificationsEmail(n, f)...)
		case "generic":
			allErrs = append(allErrs, w.validateSpecNotificationsGeneric(n, f)...)
		case "microsoft-teams":
			allErrs = append(allErrs, w.validateSpecNotificationsMicrosoftTeams(n, f)...)
		case "slack":
			allErrs = append(allErrs, w.validateSpecNotificationsSlack(n, f)...)
		}
	}

	return allErrs
}

func (w *Workspace) validateSpecNotificationsEmail(n Notification, f *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	t := "email"

	if n.Token != "" {
		allErrs = append(allErrs, field.Required(
			f,
			fmt.Sprintf("token cannot be set for type %q", t)),
		)
	}
	if n.URL != "" {
		allErrs = append(allErrs, field.Required(
			f,
			fmt.Sprintf("url cannot be set for type %q", t)),
		)
	}
	if len(n.EmailAddresses) == 0 && len(n.EmailUsers) == 0 {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			fmt.Sprintf("at least one of emailAddresses or emailUsers must be set for type %q", t)),
		)
	}

	return allErrs
}

func (w *Workspace) validateSpecNotificationsGeneric(n Notification, f *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	t := "generic"

	if n.Token == "" {
		allErrs = append(allErrs, field.Required(
			f,
			fmt.Sprintf("token must be set for type %q", t)),
		)
	}
	if n.URL == "" {
		allErrs = append(allErrs, field.Required(
			f,
			fmt.Sprintf("url must be set for type %q", t)),
		)
	}
	if len(n.EmailAddresses) != 0 {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			fmt.Sprintf("emailAddresses cannot be set for type %q", t)),
		)
	}
	if len(n.EmailUsers) != 0 {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			fmt.Sprintf("emailUsers cannot be set for type %q", t)),
		)
	}

	return allErrs
}

func (w *Workspace) validateSpecNotificationsMicrosoftTeams(n Notification, f *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	t := "microsoft-teams"

	if n.URL == "" {
		allErrs = append(allErrs, field.Required(
			f,
			fmt.Sprintf("url must be set for type %q", t)),
		)
	}
	if n.Token != "" {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			fmt.Sprintf("token cannot be set for type %q", t)),
		)
	}
	if len(n.EmailAddresses) != 0 {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			fmt.Sprintf("emailAddresses cannot be set for type %q", t)),
		)
	}
	if len(n.EmailUsers) != 0 {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			fmt.Sprintf("emailUsers cannot be set for type %q", t)),
		)
	}

	return allErrs
}

func (w *Workspace) validateSpecNotificationsSlack(n Notification, f *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	t := "slack"

	if n.URL == "" {
		allErrs = append(allErrs, field.Required(
			f,
			fmt.Sprintf("url must be set for type %q", t)),
		)
	}
	if n.Token != "" {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			fmt.Sprintf("token cannot be set for type %q", t)),
		)
	}
	if len(n.EmailAddresses) != 0 {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			fmt.Sprintf("emailAddresses cannot be set for type %q", t)),
		)
	}
	if len(n.EmailUsers) != 0 {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			fmt.Sprintf("emailUsers cannot be set for type %q", t)),
		)
	}

	return allErrs
}

func (w *Workspace) validateSpecRemoteStateSharing() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := w.Spec.RemoteStateSharing

	if spec == nil {
		return allErrs
	}

	f := field.NewPath("spec").Child("remoteStateSharing")

	if !spec.AllWorkspaces && len(spec.Workspaces) == 0 {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			"one of AllWorkspaces or Workspaces must be set: AllWorkspaces must be true or Workspaces must have at least one item"),
		)
	}

	if spec.AllWorkspaces && len(spec.Workspaces) != 0 {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			"only one of AllWorkspaces or Workspaces[] can be used at a time, not both"),
		)
	}

	if len(spec.Workspaces) != 0 {
		allErrs = append(allErrs, w.validateSpecRemoteStateSharingWorkspaces()...)
	}

	return allErrs
}

func (w *Workspace) validateSpecRemoteStateSharingWorkspaces() field.ErrorList {
	allErrs := field.ErrorList{}

	wi := make(map[string]int)
	wn := make(map[string]int)

	for i, ws := range w.Spec.RemoteStateSharing.Workspaces {
		f := field.NewPath("spec").Child("remoteStateSharing").Child(fmt.Sprintf("workspaces[%d]", i))
		if ws.ID == "" && ws.Name == "" {
			allErrs = append(allErrs, field.Invalid(
				f,
				"",
				"one of the field ID or Name must be set"),
			)
		}

		if ws.ID != "" && ws.Name != "" {
			allErrs = append(allErrs, field.Invalid(
				f,
				"",
				"only one of the field ID or Name is allowed"),
			)
		}

		if ws.ID != "" {
			if _, ok := wi[ws.ID]; ok {
				allErrs = append(allErrs, field.Duplicate(f.Child("ID"), ws.ID))
			}
			wi[ws.ID] = i
		}

		if ws.Name != "" {
			if _, ok := wn[ws.Name]; ok {
				allErrs = append(allErrs, field.Duplicate(f.Child("Name"), ws.Name))
			}
			wn[ws.Name] = i
		}
	}

	return allErrs
}

func (w *Workspace) validateSpecRunTasks() field.ErrorList {
	allErrs := field.ErrorList{}

	rti := make(map[string]int)
	rtn := make(map[string]int)

	for i, rt := range w.Spec.RunTasks {
		f := field.NewPath("spec").Child(fmt.Sprintf("runTasks[%d]", i))
		if rt.ID == "" && rt.Name == "" {
			allErrs = append(allErrs, field.Invalid(
				f,
				"",
				"one of the field ID or Name must be set"),
			)
		}

		if rt.ID != "" && rt.Name != "" {
			allErrs = append(allErrs, field.Invalid(
				f,
				"",
				"only one of the field ID or Name is allowed"),
			)
		}

		if rt.ID != "" {
			if _, ok := rti[rt.ID]; ok {
				allErrs = append(allErrs, field.Duplicate(f.Child("ID"), rt.ID))
			}
			rti[rt.ID] = i
		}

		if rt.Name != "" {
			if _, ok := rtn[rt.Name]; ok {
				allErrs = append(allErrs, field.Duplicate(f.Child("Name"), rt.Name))
			}
			rtn[rt.Name] = i
		}
	}

	return allErrs
}

func (w *Workspace) validateSpecRunTriggers() field.ErrorList {
	allErrs := field.ErrorList{}

	rti := make(map[string]int)
	rtn := make(map[string]int)

	for i, rt := range w.Spec.RunTriggers {
		f := field.NewPath("spec").Child(fmt.Sprintf("runTriggers[%d]", i))
		if rt.ID == "" && rt.Name == "" {
			allErrs = append(allErrs, field.Invalid(
				f,
				"",
				"one of the field ID or Name must be set"),
			)
		}

		if rt.ID != "" && rt.Name != "" {
			allErrs = append(allErrs, field.Invalid(
				f,
				"",
				"only one of the field ID or Name is allowed"),
			)
		}

		if rt.ID != "" {
			if _, ok := rti[rt.ID]; ok {
				allErrs = append(allErrs, field.Duplicate(f.Child("ID"), rt.ID))
			}
			rti[rt.ID] = i
		}

		if rt.Name != "" {
			if _, ok := rtn[rt.Name]; ok {
				allErrs = append(allErrs, field.Duplicate(f.Child("Name"), rt.Name))
			}
			rtn[rt.Name] = i
		}
	}

	return allErrs
}

func (w *Workspace) validateSpecSSHKey() field.ErrorList {
	allErrs := field.ErrorList{}
	spec := w.Spec.SSHKey

	if spec == nil {
		return allErrs
	}

	f := field.NewPath("spec").Child("sshKey")

	if spec.ID == "" && spec.Name == "" {
		allErrs = append(allErrs, field.Invalid(
			f,
			"",
			"one of the field ID or Name must be set"),
		)
	}

	if w.Spec.SSHKey.ID != "" && w.Spec.SSHKey.Name != "" {
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
// + EnvironmentVariables names duplicate: spec.environmentVariables[].name
// + TerraformVariables names duplicate: spec.terraformVariables[].name
// + Tags duplicate: spec.tags[]
// + AgentPool must be set when ExecutionMode = 'agent': spec.agentPool <- spec.executionMode['agent']
//
// + Invalid CR cannot be deleted until it is fixed -- need to discuss if we want to do something about it
