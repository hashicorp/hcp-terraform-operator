// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	tfc "github.com/hashicorp/go-tfe"
)

var runStatuses = map[tfc.RunStatus]struct{}{
	tfc.RunApplied:                  {},
	tfc.RunApplying:                 {},
	tfc.RunApplyQueued:              {},
	tfc.RunCanceled:                 {},
	tfc.RunConfirmed:                {},
	tfc.RunCostEstimated:            {},
	tfc.RunCostEstimating:           {},
	tfc.RunDiscarded:                {},
	tfc.RunErrored:                  {},
	tfc.RunFetching:                 {},
	tfc.RunFetchingCompleted:        {},
	tfc.RunPending:                  {},
	tfc.RunPlanned:                  {},
	tfc.RunPlannedAndFinished:       {},
	tfc.RunPlannedAndSaved:          {},
	tfc.RunPlanning:                 {},
	tfc.RunPlanQueued:               {},
	tfc.RunPolicyChecked:            {},
	tfc.RunPolicyChecking:           {},
	tfc.RunPolicyOverride:           {},
	tfc.RunPolicySoftFailed:         {},
	tfc.RunPostPlanAwaitingDecision: {},
	tfc.RunPostPlanCompleted:        {},
	tfc.RunPostPlanRunning:          {},
	tfc.RunPreApplyRunning:          {},
	tfc.RunPreApplyCompleted:        {},
	tfc.RunPrePlanCompleted:         {},
	tfc.RunPrePlanRunning:           {},
	tfc.RunQueuing:                  {},
	tfc.RunQueuingApply:             {},
}

func TestRunCompleted(t *testing.T) {
	t.Parallel()

	trueCases := map[tfc.RunStatus]struct{}{
		tfc.RunApplied:            {},
		tfc.RunCanceled:           {},
		tfc.RunDiscarded:          {},
		tfc.RunErrored:            {},
		tfc.RunPlannedAndFinished: {},
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

	trueCases := map[tfc.RunStatus]struct{}{
		tfc.RunApplied:            {},
		tfc.RunPlannedAndFinished: {},
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
