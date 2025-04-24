// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// Agent Pool Metrics
var (
	metricLabels = []string{
		"id",
		"name",
	}
	metricDesiredAgents = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hcp_tf_desired_agents",
			Help: "Number of desired agents",
		},
		metricLabels,
	)
	metricRequiredAgents = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hcp_tf_required_agents",
			Help: "Number of required agents",
		},
		metricLabels,
	)
	metricMinAgents = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hcp_tf_min_agents",
		},
		metricLabels,
	)
	metricMaxAgents = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hcp_tf_max_agents",
		},
		metricLabels,
	)
	metricConnectedWorkspaces = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hcp_tf_connected_workspaces",
		},
		metricLabels,
	)
)

func RegisterMetrics() {
	metrics.Registry.MustRegister(
		metricRequiredAgents,
		metricDesiredAgents,
		metricMinAgents,
		metricMaxAgents,
		metricConnectedWorkspaces,
	)
}
