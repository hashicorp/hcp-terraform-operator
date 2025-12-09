// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	tfc "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/go-tfe/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func TestpendingRuns(t *testing.T) {
	tests := []struct {
		name          string
		mockRuns      []*tfc.Run
		mockErr       error
		expectedCount int32
		expectError   bool
	}{
		{
			name:          "returns error from client",
			mockErr:       errors.New("api error"),
			expectedCount: 0,
			expectError:   true,
		},
		{
			name: "counts plan-only runs",
			mockRuns: []*tfc.Run{
				{ID: "run1", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run2", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws2"}},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "skips user interaction runs",
			mockRuns: []*tfc.Run{
				{ID: "run1", PlanOnly: false, Status: tfc.RunPlanned, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run2", PlanOnly: false, Status: tfc.RunPolicyOverride, Workspace: &tfc.Workspace{ID: "ws2"}},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "counts normal pending runs",
			mockRuns: []*tfc.Run{
				{ID: "run1", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run2", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws2"}},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "mix of plan-only and normal runs",
			mockRuns: []*tfc.Run{
				{ID: "run1", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run2", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws2"}},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "plan-only runs for single workspace",
			mockRuns: []*tfc.Run{
				{ID: "run1", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run2", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run3", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run4", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
			},
			expectedCount: 4,
			expectError:   false,
		},
		{
			name: "single apply and multiple plan-only runs for single workspace",
			mockRuns: []*tfc.Run{
				{ID: "run1", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run2", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run3", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run4", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run5", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
			},
			expectedCount: 5,
			expectError:   false,
		},
		{
			name: "mix of plan-only and apply runs for single workspace",
			mockRuns: []*tfc.Run{
				{ID: "run1", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run2", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run3", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run4", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run5", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
			},
			expectedCount: 4,
			expectError:   false,
		},
		{
			name: "mix of plan-only and apply runs for multiple workspaces",
			mockRuns: []*tfc.Run{
				{ID: "run1", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run2", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws2"}},
				{ID: "run3", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws3"}},
				{ID: "run4", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run5", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
			},
			expectedCount: 5,
			expectError:   false,
		},
		{
			name: "mix of plan-only and apply runs for two workspaces",
			mockRuns: []*tfc.Run{
				{ID: "run1", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws1"}},
				{ID: "run2", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws2"}},
				{ID: "run3", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws2"}},
				{ID: "run4", PlanOnly: true, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws2"}},
				{ID: "run5", PlanOnly: false, Status: tfc.RunPlanning, Workspace: &tfc.Workspace{ID: "ws2"}},
			},
			expectedCount: 4,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRuns := mocks.NewMockRuns(ctrl)
			mockRuns.EXPECT().
				ListForOrganization(gomock.Any(), "test-org", gomock.Any()).
				Return(&tfc.OrganizationRunList{Items: tt.mockRuns, PaginationNextPrev: &tfc.PaginationNextPrev{NextPage: 0}}, tt.mockErr)

			ap := &agentPoolInstance{
				tfClient: HCPTerraformClient{Client: &tfc.Client{Runs: mockRuns}},
				instance: appv1alpha2.AgentPool{
					Spec: appv1alpha2.AgentPoolSpec{
						Name:         "test-pool",
						Organization: "test-org",
					},
				},
				log: logr.Logger{},
			}

			count, err := pendingRuns(context.Background(), ap)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, count)
			}
		})
	}
}
