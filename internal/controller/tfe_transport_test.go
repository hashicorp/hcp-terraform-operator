// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	tfc "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/stretchr/testify/require"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func runJSON(id, status string) string {
	return fmt.Sprintf(`{"id":%q,"type":"runs","attributes":{"status":%q,"plan-only":false}}`, id, status)
}

func TestRunStatusTransport(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch r.URL.Path {
		case "/api/v2/runs/run-await":
			fmt.Fprintf(w, `{"data":%s}`, runJSON("run-await", "post_plan_awaiting_decision"))
		case "/api/v2/runs/run-planned":
			fmt.Fprintf(w, `{"data":%s}`, runJSON("run-planned", "planned"))
		case "/api/v2/organizations/org/runs":
			fmt.Fprintf(w, `{"data":[%s,%s]}`, runJSON("run-await", "post_plan_awaiting_decision"), runJSON("run-planned", "planned"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := tfc.NewClient(&tfc.Config{
		Address:       srv.URL,
		Token:         "token",
		HTTPTransport: newRunStatusTransport(http.DefaultTransport),
	})
	require.NoError(t, err)

	ctx := context.Background()

	for id, want := range map[string]string{
		"run-await":   appv1alpha2.RunPostPlanAwaitingDecision,
		"run-planned": models.PLANNED_RUNS_ATTRIBUTES_STATUS.String(),
	} {
		resp, err := client.API.Runs().ById(id).Get(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, want, runAttributesStatus(resp.GetData().GetAttributes()), id)
	}

	list, err := client.API.Organizations().ByOrganization_name("org").Runs().Get(ctx, nil)
	require.NoError(t, err)
	got := []string{}
	for _, r := range list.GetData() {
		got = append(got, runAttributesStatus(r.GetAttributes()))
	}
	require.Equal(t, []string{appv1alpha2.RunPostPlanAwaitingDecision, models.PLANNED_RUNS_ATTRIBUTES_STATUS.String()}, got)
}
