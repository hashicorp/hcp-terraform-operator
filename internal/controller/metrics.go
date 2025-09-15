// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
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
		[]string{
			"run_status",
		},
	)
	metricRunsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hcp_tf_runs_total",
			Help: "HCP Terraform - Total number of pending Runs by statuses",
		},
		[]string{},
	)
)

func RegisterMetrics() {
	metrics.Registry.MustRegister(
		metricRuns,
	)
}
