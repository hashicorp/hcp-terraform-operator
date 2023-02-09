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
	allErrs = append(allErrs, w.validateSpecRemoteStateSharing()...)
	allErrs = append(allErrs, w.validateSpecRunTrigger()...)
	allErrs = append(allErrs, w.validateSpecSSHKey()...)

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: GroupVersion.Group, Kind: "Module"},
		w.Name,
		allErrs,
	)
}

func (w *Workspace) validateSpecAgentPool() field.ErrorList {
	allErrs := field.ErrorList{}

	if w.Spec.AgentPool == nil {
		return allErrs
	}

	if w.Spec.AgentPool.ID == "" && w.Spec.AgentPool.Name == "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec"),
			"agentPool",
			"one of ID or Name must be set"),
		)
	}

	if w.Spec.AgentPool.ID != "" && w.Spec.AgentPool.Name != "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec"),
			"agentPool",
			"only one of ID or Name can be used at a time, not both"),
		)
	}

	return allErrs
}

func (w *Workspace) validateSpecRemoteStateSharing() field.ErrorList {
	allErrs := field.ErrorList{}

	if w.Spec.RemoteStateSharing == nil {
		return allErrs
	}

	if !w.Spec.RemoteStateSharing.AllWorkspaces && len(w.Spec.RemoteStateSharing.Workspaces) == 0 {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec"),
			"remoteStateSharing",
			"one of AllWorkspaces or Workspaces must be set: AllWorkspaces must be true or Workspaces length must not be 0"),
		)
	}

	if w.Spec.RemoteStateSharing.AllWorkspaces && len(w.Spec.RemoteStateSharing.Workspaces) != 0 {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec"),
			"agentPool",
			"only one of AllWorkspaces or Workspaces[] can be used at a time, not both"),
		)
	}

	if len(w.Spec.RemoteStateSharing.Workspaces) != 0 {
		allErrs = append(allErrs, w.validateSpecRemoteStateSharingWorkspaces()...)
	}

	return allErrs
}

func (w *Workspace) validateSpecRemoteStateSharingWorkspaces() field.ErrorList {
	allErrs := field.ErrorList{}

	wi := make(map[string]int)
	wn := make(map[string]int)

	for i, ws := range w.Spec.RemoteStateSharing.Workspaces {
		if ws.ID == "" && ws.Name == "" {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("remoteStateSharing"),
				fmt.Sprintf("workspaces[%d]", i),
				"one of ID or Name must be set"),
			)
		}

		if ws.ID != "" && ws.Name != "" {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec").Child("remoteStateSharing"),
				fmt.Sprintf("workspaces[%d]", i),
				"only one of ID or Name can be used at a time, not both"),
			)
		}

		if ws.ID != "" {
			if v, ok := wi[ws.ID]; ok {
				allErrs = append(allErrs, field.Duplicate(
					field.NewPath("spec").Child("remoteStateSharing"),
					fmt.Sprintf("workspaces.id[%d] is the same as workspaces.id[%d]: %s", i, v, ws.ID),
				))
			}
			wi[ws.ID] = i
		}

		if ws.Name != "" {
			if v, ok := wn[ws.Name]; ok {
				allErrs = append(allErrs, field.Duplicate(
					field.NewPath("spec").Child("remoteStateSharing"),
					fmt.Sprintf("workspaces.name[%d] is the same as workspaces.name[%d]: %s", i, v, ws.Name),
				))
			}
			wn[ws.Name] = i
		}
	}

	return allErrs
}

func (w *Workspace) validateSpecRunTrigger() field.ErrorList {
	allErrs := field.ErrorList{}

	rti := make(map[string]int)
	rtn := make(map[string]int)

	for i, rt := range w.Spec.RunTriggers {
		if rt.ID == "" && rt.Name == "" {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec"),
				fmt.Sprintf("runTriggers[%d]", i),
				"one of ID or Name must be set"),
			)
		}

		if rt.ID != "" && rt.Name != "" {
			allErrs = append(allErrs, field.Invalid(
				field.NewPath("spec"),
				fmt.Sprintf("runTriggers[%d]", i),
				"only one of ID or Name can be used at a time, not both"),
			)
		}

		if rt.ID != "" {
			if v, ok := rti[rt.ID]; ok {
				allErrs = append(allErrs, field.Duplicate(
					field.NewPath("spec").Child("remoteStateSharing"),
					fmt.Sprintf("workspaces.id[%d] is the same as workspaces.id[%d]: %s", i, v, rt.ID),
				))
			}
			rti[rt.ID] = i
		}

		if rt.Name != "" {
			if v, ok := rtn[rt.Name]; ok {
				allErrs = append(allErrs, field.Duplicate(
					field.NewPath("spec").Child("remoteStateSharing"),
					fmt.Sprintf("workspaces.name[%d] is the same as workspaces.name[%d]: %s", i, v, rt.Name),
				))
			}
			rtn[rt.Name] = i
		}
	}

	return allErrs
}

func (w *Workspace) validateSpecSSHKey() field.ErrorList {
	allErrs := field.ErrorList{}

	if w.Spec.SSHKey == nil {
		return allErrs
	}

	if w.Spec.SSHKey.ID == "" && w.Spec.SSHKey.Name == "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec"),
			"sshKey",
			"one of ID or Name must be set"),
		)
	}

	if w.Spec.SSHKey.ID != "" && w.Spec.SSHKey.Name != "" {
		allErrs = append(allErrs, field.Invalid(
			field.NewPath("spec"),
			"SSHKey",
			"only one of ID or Name can be used at a time, not both"),
		)
	}

	return allErrs
}
