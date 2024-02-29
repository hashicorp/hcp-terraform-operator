// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	tfc "github.com/hashicorp/go-tfe"
)

func (rs *RunStatus) RunCompleted() bool {
	return runCompleted(rs.Status)
}

// runCompleted returns true if the run is completed
func runCompleted(status string) bool {
	// The following Run statuses indicate the completion
	switch status {
	case string(tfc.RunApplied):
		return true
	case string(tfc.RunPlannedAndFinished):
		return true
	case string(tfc.RunErrored):
		return true
	case string(tfc.RunCanceled):
		return true
	case string(tfc.RunDiscarded):
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
	case string(tfc.RunApplied):
		return true
	case string(tfc.RunPlannedAndFinished):
		return true
	}

	return false
}
