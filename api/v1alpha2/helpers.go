// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import "github.com/hashicorp/go-tfe/v2/api/models"

// RunPostPlanAwaitingDecision is a run status that is missing from the go-tfe v2 status enum.
const RunPostPlanAwaitingDecision = "post_plan_awaiting_decision"

func (rs *RunStatus) RunCompleted() bool {
	return runCompleted(rs.Status)
}

func (rs *PlanStatus) RunCompleted() bool {
	return runCompleted(rs.Status)
}

func runCompleted(status string) bool {
	// The following Run statuses indicate the completion
	switch status {
	case models.APPLIED_RUNS_ATTRIBUTES_STATUS.String():
		return true
	case models.PLANNED_AND_FINISHED_RUNS_ATTRIBUTES_STATUS.String():
		return true
	case models.ERRORED_RUNS_ATTRIBUTES_STATUS.String():
		return true
	case models.CANCELED_RUNS_ATTRIBUTES_STATUS.String():
		return true
	case models.DISCARDED_RUNS_ATTRIBUTES_STATUS.String():
		return true
	}

	return false
}

func (rs *RunStatus) RunApplied() bool {
	return runApplied(rs.Status)
}

// runApplied returns true if the run is applied
func runApplied(status string) bool {
	// The following Run statuses indicate the completion
	switch status {
	case models.APPLIED_RUNS_ATTRIBUTES_STATUS.String():
		return true
	case models.PLANNED_AND_FINISHED_RUNS_ATTRIBUTES_STATUS.String():
		return true
	}

	return false
}
