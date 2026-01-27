// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

func (s *Stack) IsCreationCandidate() bool {
	return s.Status.StackID == ""
}

// AddOrUpdateDeploymentStatus adds a given deployment to the status if it does not exist there; otherwise, it updates it.
func (st *StackStatus) AddOrUpdateDeploymentStatus(deployment DeploymentStatus) {
	for i, d := range st.Deployments {
		if d.Name == deployment.Name {
			st.Deployments[i].ID = deployment.ID
			st.Deployments[i].Status = deployment.Status
			st.Deployments[i].UpdatedAt = deployment.UpdatedAt
			return
		}
	}

	st.Deployments = append(st.Deployments, DeploymentStatus{
		Name:      deployment.Name,
		ID:        deployment.ID,
		Status:    deployment.Status,
		UpdatedAt: deployment.UpdatedAt,
	})
}

// GetDeploymentStatus returns a given deployment from the status if it exists there; otherwise, nil.
func (st *StackStatus) GetDeploymentStatus(name string) *DeploymentStatus {
	for _, d := range st.Deployments {
		if d.Name == name {
			return &d
		}
	}

	return nil
}

// DeleteDeploymentStatus deletes a given deployment from the status.
func (st *StackStatus) DeleteDeploymentStatus(name string) {
	for i, d := range st.Deployments {
		if d.Name == name {
			st.Deployments = append(st.Deployments[:i], st.Deployments[i+1:]...)
			return
		}
	}
}

// Made with Bob
