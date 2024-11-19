// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package unit

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestRBACClusterRoleProxyCreateTrue(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "true",
		},
		Version: helmChartVersion,
	}
	rbac := renderRBACClusterRoleProxyManifest(t, options)

	assert.Equal(t, defaultRBACClusterRoleProxyName, rbac.Name)
	assert.Empty(t, rbac.Labels)
	assert.Empty(t, rbac.Annotations)

	testRBACClusterRoleProxyRules(t, rbac)
}

func TestRBACClusterRoleProxyCreateFalse(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "false",
		},
		Version: helmChartVersion,
	}
	_, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacClusterRoleProxyTemplate})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find template")
}

func testRBACClusterRoleProxyRules(t *testing.T, rbac rbacv1.ClusterRole) {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"authentication.k8s.io"},
			Resources: []string{"tokenreviews"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"authorization.k8s.io"},
			Resources: []string{"subjectaccessreviews"},
			Verbs:     []string{"create"},
		},
	}
	assert.Equal(t, rules, rbac.Rules)
}
