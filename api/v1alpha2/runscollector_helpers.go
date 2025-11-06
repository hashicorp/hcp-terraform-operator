// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

func (rc *RunsCollector) NeedUpdateStatus() bool {
	if rc.Status.AgentPool == nil {
		return true
	}
	if rc.Spec.AgentPool.Name != "" {
		return rc.Spec.AgentPool.Name != rc.Status.AgentPool.Name
	}
	if rc.Spec.AgentPool.ID != "" {
		return rc.Spec.AgentPool.ID != rc.Status.AgentPool.ID
	}

	return false
}
