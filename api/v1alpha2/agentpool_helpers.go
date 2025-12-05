// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

func (ap *AgentPool) IsCreationCandidate() bool {
	return ap.Status.AgentPoolID == ""
}
