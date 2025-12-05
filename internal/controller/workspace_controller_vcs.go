// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

// getTriggerPatterns return a map that maps consist of all trigger patterns defined in a object specification
// and values 'true' to simulate the Set structure.
func getTriggerPatterns(instance *appv1alpha2.Workspace) map[string]struct{} {
	patterns := make(map[string]struct{})

	if instance.Spec.VersionControl == nil || len(instance.Spec.VersionControl.TriggerPatterns) == 0 {
		return patterns
	}

	for _, t := range instance.Spec.VersionControl.TriggerPatterns {
		patterns[string(t)] = struct{}{}
	}

	return patterns
}

// getWorkspaceTriggerPatterns return a map that maps consist of all trigger patterns assigned to workspace
// and values 'true' to simulate the Set structure.
func getWorkspaceTriggerPatterns(workspace *tfc.Workspace) map[string]struct{} {
	patterns := make(map[string]struct{})

	if len(workspace.TriggerPatterns) == 0 {
		return patterns
	}

	for _, t := range workspace.TriggerPatterns {
		patterns[t] = struct{}{}
	}

	return patterns
}

// getTriggerPrefixes return a map that maps consist of all trigger prefixes defined in a object specification
// and values 'true' to simulate the Set structure.
func getTriggerPrefixes(instance *appv1alpha2.Workspace) map[string]struct{} {
	prefixes := make(map[string]struct{})

	if instance.Spec.VersionControl == nil || len(instance.Spec.VersionControl.TriggerPrefixes) == 0 {
		return prefixes
	}

	for _, t := range instance.Spec.VersionControl.TriggerPrefixes {
		prefixes[string(t)] = struct{}{}
	}

	return prefixes
}

// getWorkspaceTriggerPrefixes return a map that maps consist of all trigger prefixes assigned to workspace
// and values 'true' to simulate the Set structure.
func getWorkspaceTriggerPrefixes(workspace *tfc.Workspace) map[string]struct{} {
	prefixes := make(map[string]struct{})

	if len(workspace.TriggerPrefixes) == 0 {
		return prefixes
	}

	for _, t := range workspace.TriggerPrefixes {
		prefixes[t] = struct{}{}
	}

	return prefixes
}

// vcsTriggersDifference returns the list of file trigger prefixes that consists of the elements of leftTriggers
// which are not elements of rightTriggers.
func vcsTriggersDifference(leftTriggers, rightTriggers map[string]struct{}) []string {
	var d []string

	for t := range leftTriggers {
		if _, ok := rightTriggers[t]; !ok {
			d = append(d, t)
		}
	}

	return d
}
