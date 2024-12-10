// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package unit

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestRBACClusterRoleMetricsReaderCreateTrue(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "true",
		},
		Version: helmChartVersion,
	}
	rbac := renderRBACClusterRoleMetricsReaderManifest(t, options)

	assert.Equal(t, defaultRBACClusterRoleMetricsReaderName, rbac.Name)
	assert.Empty(t, rbac.Labels)
	assert.Empty(t, rbac.Annotations)

	testRBACClusterRoleMetricsReaderRules(t, rbac)
}

func TestRBACClusterRoleMetricsReaderCreateFalse(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "false",
		},
		Version: helmChartVersion,
	}
	_, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacClusterRoleMetricsReaderTemplate})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find template")
}

func testRBACClusterRoleMetricsReaderRules(t *testing.T, rbac rbacv1.ClusterRole) {
	rules := []rbacv1.PolicyRule{
		{
			NonResourceURLs: []string{"/metrics"},
			Verbs:           []string{"get"},
		},
	}
	assert.Equal(t, rules, rbac.Rules)
}
