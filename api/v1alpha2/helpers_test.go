// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	"github.com/hashicorp/go-tfe/v2/api/models"
)

var runStatuses = map[string]struct{}{
	models.APPLIED_RUNS_ATTRIBUTES_STATUS.String():              {},
	models.APPLYING_RUNS_ATTRIBUTES_STATUS.String():             {},
	models.APPLY_QUEUED_RUNS_ATTRIBUTES_STATUS.String():         {},
	models.CANCELED_RUNS_ATTRIBUTES_STATUS.String():             {},
	models.CONFIRMED_RUNS_ATTRIBUTES_STATUS.String():            {},
	models.COST_ESTIMATED_RUNS_ATTRIBUTES_STATUS.String():       {},
	models.COST_ESTIMATING_RUNS_ATTRIBUTES_STATUS.String():      {},
	models.DISCARDED_RUNS_ATTRIBUTES_STATUS.String():            {},
	models.ERRORED_RUNS_ATTRIBUTES_STATUS.String():              {},
	models.FETCHING_RUNS_ATTRIBUTES_STATUS.String():             {},
	models.FETCHING_COMPLETED_RUNS_ATTRIBUTES_STATUS.String():   {},
	models.PENDING_RUNS_ATTRIBUTES_STATUS.String():              {},
	models.PLANNED_RUNS_ATTRIBUTES_STATUS.String():              {},
	models.PLANNED_AND_FINISHED_RUNS_ATTRIBUTES_STATUS.String(): {},
	models.PLANNED_AND_SAVED_RUNS_ATTRIBUTES_STATUS.String():    {},
	models.PLANNING_RUNS_ATTRIBUTES_STATUS.String():             {},
	models.PLAN_QUEUED_RUNS_ATTRIBUTES_STATUS.String():          {},
	models.POLICY_CHECKED_RUNS_ATTRIBUTES_STATUS.String():       {},
	models.POLICY_CHECKING_RUNS_ATTRIBUTES_STATUS.String():      {},
	models.POLICY_OVERRIDE_RUNS_ATTRIBUTES_STATUS.String():      {},
	models.POLICY_SOFT_FAILED_RUNS_ATTRIBUTES_STATUS.String():   {},
	RunPostPlanAwaitingDecision:                                 {},
	models.POST_PLAN_COMPLETED_RUNS_ATTRIBUTES_STATUS.String():  {},
	models.POST_PLAN_RUNNING_RUNS_ATTRIBUTES_STATUS.String():    {},
	models.PRE_APPLY_RUNNING_RUNS_ATTRIBUTES_STATUS.String():    {},
	models.PRE_APPLY_COMPLETED_RUNS_ATTRIBUTES_STATUS.String():  {},
	models.PRE_PLAN_COMPLETED_RUNS_ATTRIBUTES_STATUS.String():   {},
	models.PRE_PLAN_RUNNING_RUNS_ATTRIBUTES_STATUS.String():     {},
	models.QUEUING_RUNS_ATTRIBUTES_STATUS.String():              {},
	models.QUEUING_APPLY_RUNS_ATTRIBUTES_STATUS.String():        {},
}

func TestRunCompleted(t *testing.T) {
	t.Parallel()

	trueCases := map[string]struct{}{
		models.APPLIED_RUNS_ATTRIBUTES_STATUS.String():              {},
		models.CANCELED_RUNS_ATTRIBUTES_STATUS.String():             {},
		models.DISCARDED_RUNS_ATTRIBUTES_STATUS.String():            {},
		models.ERRORED_RUNS_ATTRIBUTES_STATUS.String():              {},
		models.PLANNED_AND_FINISHED_RUNS_ATTRIBUTES_STATUS.String(): {},
	}

	for n := range runStatuses {
		t.Run(string(n), func(t *testing.T) {
			if runCompleted(string(n)) {
				if _, ok := trueCases[n]; !ok {
					t.Fatalf("Expected result to be false but got true for status %#v", n)
				}
			}
		})
	}
}

func TestRunApplied(t *testing.T) {
	t.Parallel()

	trueCases := map[string]struct{}{
		models.APPLIED_RUNS_ATTRIBUTES_STATUS.String():              {},
		models.PLANNED_AND_FINISHED_RUNS_ATTRIBUTES_STATUS.String(): {},
	}

	for n := range runStatuses {
		t.Run(string(n), func(t *testing.T) {
			if runApplied(string(n)) {
				if _, ok := trueCases[n]; !ok {
					t.Fatalf("Expected result to be false but got true for status %#v", n)
				}
			}
		})
	}
}
