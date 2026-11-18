// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-logr/logr"
	tfc "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func TestPendingRuns(t *testing.T) {
	tests := []struct {
		name          string
		mockRuns      []testRun
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
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "skips user interaction runs",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: false, Status: models.PLANNED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: false, Status: models.POLICY_OVERRIDE_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
			},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name: "counts normal pending runs",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "mix of plan-only and normal runs",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
			},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "plan-only runs for single workspace",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run3", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run4", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
			},
			expectedCount: 4,
			expectError:   false,
		},
		{
			name: "single apply and multiple plan-only runs for single workspace",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run3", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run4", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run5", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
			},
			expectedCount: 5,
			expectError:   false,
		},
		{
			name: "mix of plan-only and apply runs for single workspace",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run3", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run4", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run5", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
			},
			expectedCount: 4,
			expectError:   false,
		},
		{
			name: "mix of plan-only and apply runs for multiple workspaces",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run3", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws3"},
				{ID: "run4", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run5", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
			},
			expectedCount: 5,
			expectError:   false,
		},
		{
			name: "mix of plan-only and apply runs for two workspaces",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run3", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run4", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run5", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
			},
			expectedCount: 4,
			expectError:   false,
		},
		{
			name: "plan-only runs that might have a pending status",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PENDING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run2", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run3", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run4", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run5", PlanOnly: true, Status: models.PENDING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "mix of plan-only and apply runs that might have user interaction statuses",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: false, Status: models.PLANNED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run3", PlanOnly: true, Status: models.PENDING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run4", PlanOnly: false, Status: models.COST_ESTIMATED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run5", PlanOnly: false, Status: models.PENDING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
			},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name: "plan and apply runs that might have user interaction status PolicyChecked",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PENDING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run2", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run3", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run4", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run5", PlanOnly: true, Status: models.POLICY_CHECKED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
			},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name: "skips terminal plan-only runs (planned_and_finished)",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PLANNED_AND_FINISHED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
			},
			expectedCount: 1, // only the active run counts
		},
		{
			name: "skips all terminal plan-only run statuses",
			mockRuns: []testRun{
				{ID: "run1", PlanOnly: true, Status: models.PLANNED_AND_FINISHED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run2", PlanOnly: true, Status: models.ERRORED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"},
				{ID: "run3", PlanOnly: true, Status: models.CANCELED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws3"},
				{ID: "run4", PlanOnly: true, Status: models.DISCARDED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws4"},
				{ID: "run5", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws5"},
				{ID: "run6", PlanOnly: false, Status: models.COST_ESTIMATED_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
				{ID: "run7", PlanOnly: false, Status: models.PENDING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"},
			},
			expectedCount: 1, // only the active run counts
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v2/organizations/test-org/runs" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if tt.mockErr != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/vnd.api+json")
				_, _ = w.Write(runsResponse(tt.mockRuns, 0))
			}))
			defer srv.Close()

			ap := newTestAgentPoolInstance(t, srv.URL)

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

func TestPendingRunsQueryAndPagination(t *testing.T) {
	var queries []url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query())
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if r.URL.Query().Get("page[number]") == "1" {
			_, _ = w.Write(runsResponse([]testRun{{ID: "run1", PlanOnly: true, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws1"}}, 2))
			return
		}
		_, _ = w.Write(runsResponse([]testRun{{ID: "run2", PlanOnly: false, Status: models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(), WorkspaceID: "ws2"}}, 0))
	}))
	defer srv.Close()

	count, err := pendingRuns(context.Background(), newTestAgentPoolInstance(t, srv.URL))
	assert.NoError(t, err)
	assert.Equal(t, int32(2), count)
	if assert.Len(t, queries, 2) {
		assert.Equal(t, "test-pool", queries[0].Get("filter[agent_pool_names]"))
		assert.Equal(t, "non_final", queries[0].Get("filter[status_group]"))
		assert.Equal(t, "100", queries[0].Get("page[size]"))
		assert.Equal(t, "1", queries[0].Get("page[number]"))
		assert.Equal(t, "2", queries[1].Get("page[number]"))
	}
}

type testRun struct {
	ID          string
	PlanOnly    bool
	Status      string
	WorkspaceID string
}

func runsResponse(runs []testRun, nextPage int) []byte {
	data := make([]map[string]any, 0, len(runs))
	for _, r := range runs {
		data = append(data, map[string]any{
			"id":   r.ID,
			"type": "runs",
			"attributes": map[string]any{
				"status":    string(r.Status),
				"plan-only": r.PlanOnly,
			},
			"relationships": map[string]any{
				"workspace": map[string]any{
					"data": map[string]any{"id": r.WorkspaceID, "type": "workspaces"},
				},
			},
		})
	}
	pagination := map[string]any{"current-page": 1, "page-size": 100, "prev-page": nil, "next-page": nil, "total-pages": 1, "total-count": len(runs)}
	if nextPage != 0 {
		pagination["next-page"] = nextPage
	}
	b, _ := json.Marshal(map[string]any{
		"data": data,
		"meta": map[string]any{"pagination": pagination},
	})
	return b
}

func newTestAgentPoolInstance(t *testing.T, address string) *agentPoolInstance {
	t.Helper()

	client, err := tfc.NewClient(&tfc.Config{Address: address, Token: "test-token"})
	require.NoError(t, err)

	return &agentPoolInstance{
		tfClient: HCPTerraformClient{Client: client},
		instance: appv1alpha2.AgentPool{
			Spec: appv1alpha2.AgentPoolSpec{
				Name:         "test-pool",
				Organization: "test-org",
			},
		},
		log: logr.Logger{},
	}
}

func TestComputeRequiredAgents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/organizations/test-org/workspaces" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		assert.Equal(t, "plan_queued,apply_queued,applying,planning", r.URL.Query().Get("filter[current-run][status]"))
		ws := func(id, name, pool string) map[string]any {
			return map[string]any{
				"id": id, "type": "workspaces",
				"attributes": map[string]any{"name": name},
				"relationships": map[string]any{
					"agent-pool": map[string]any{"data": map[string]any{"id": pool, "type": "agent-pools"}},
				},
			}
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{ws("ws-1", "a", "apool-1"), ws("ws-2", "b", "apool-1"), ws("ws-3", "c", "apool-2")},
			"meta": map[string]any{"pagination": map[string]any{"current-page": 1, "next-page": nil}},
		})
	}))
	defer srv.Close()

	ap := newTestAgentPoolInstance(t, srv.URL)
	ap.instance.Status.AgentPoolID = "apool-1"
	ap.instance.Spec.AgentDeploymentAutoscaling = &appv1alpha2.AgentDeploymentAutoscaling{}

	n, err := computeRequiredAgents(context.Background(), ap)
	require.NoError(t, err)
	assert.Equal(t, int32(2), n)
}
