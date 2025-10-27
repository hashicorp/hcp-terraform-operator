// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Runs Metrics
var (
	metricRuns = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hcp_tf_runs",
			Help: "HCP Terraform - Pending runs by statuses",
		},
		// TODO:
		// - Add a status label to indicate whether the metric is up or down
		//   (for example, when an endpoint is unreachable or the CR is suspended).
		// - Add agent_pool_name as label.
		// - Add agent_pool_id as label.
		[]string{
			"run_status",
		},
	)
	metricRunsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hcp_tf_runs_total",
			Help: "HCP Terraform - Total number of pending Runs by statuses",
		},
		// TODO:
		// - Add a status label to indicate whether the metric is up or down
		//   (for example, when an endpoint is unreachable or the CR is suspended).
		// - Add agent_pool_name as label.
		// - Add agent_pool_id as label.
		[]string{},
	)
	// TODO:
	// - Add a metric to track associated Workspaces.
)

func RegisterMetrics() {
	metrics.Registry.MustRegister(
		metricRuns,
		metricRunsTotal,
	)
	// Initialize all run status metrics to 0.
	for _, s := range runStatuses {
		metricRuns.WithLabelValues(string(s)).Set(0)
	}
	// Initialize total runs metric to 0.
	metricRunsTotal.WithLabelValues().Set(0)
}

func ListHCPTMetrics() error {
	mfs, err := metrics.Registry.Gather()
	if err != nil {
		return err
	}
	for _, mf := range mfs {
		name := mf.GetName()
		if strings.HasPrefix(name, "hcp_tf") {
			fmt.Printf("# HELP %s %s\n", name, mf.GetHelp())
			fmt.Printf("# TYPE %s %s\n", name, mf.Type.String())
		}
	}

	return nil
}
