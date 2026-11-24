// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

// IsCreationCandidate reports whether the controller should attempt to CREATE a new agent pool.
// False whenever status already has an ID, or when this CR is bound to a pre-existing pool via
// spec.existingAgentPoolID -- in that case it should be adopted, never created.
func (ap *AgentPool) IsCreationCandidate() bool {
	return ap.Status.AgentPoolID == "" && ap.Spec.ExistingAgentPoolID == ""
}

// IsExistingPoolReference reports whether this CR should adopt a pool that already exists,
// identified by spec.existingAgentPoolID, rather than ever creating one of its own.
func (ap *AgentPool) IsExistingPoolReference() bool {
	return ap.Spec.ExistingAgentPoolID != ""
}
