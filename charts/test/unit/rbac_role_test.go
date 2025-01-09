// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package unit

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestRBACRoleCreateTrue(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "true",
		},
		Version: helmChartVersion,
	}
	rbac := renderRBACRoleManifest(t, options)

	assert.Equal(t, defaultRBACRoleName, rbac.Name)
	assert.Empty(t, rbac.Labels)
	assert.Empty(t, rbac.Annotations)
	assert.Equal(t, defaultNamespace, rbac.Namespace)

	testRBACRoleRules(t, rbac)
}

func TestRBACRoleCreateFalse(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "false",
		},
		Version: helmChartVersion,
	}
	_, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacRoleTemplate})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find template")
}

func TestRBACRoleNamespace(t *testing.T) {
	ns := "this"
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "true",
		},
		EnvVars: map[string]string{
			"HELM_NAMESPACE": ns,
		},
		Version: helmChartVersion,
	}
	rbac := renderRBACRoleManifest(t, options)

	assert.Equal(t, defaultRBACRoleName, rbac.Name)
	assert.Empty(t, rbac.Labels)
	assert.Empty(t, rbac.Annotations)
	assert.Equal(t, ns, rbac.Namespace)

	testRBACRoleRules(t, rbac)
}

func testRBACRoleRules(t *testing.T, rbac rbacv1.Role) {
	rules := []rbacv1.PolicyRule{
		{
			Verbs: []string{
				"get",
				"list",
				"watch",
				"create",
				"update",
				"patch",
				"delete",
			},
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
		},
		{
			Verbs: []string{
				"get",
				"list",
				"watch",
				"create",
				"update",
				"patch",
				"delete",
			},
			APIGroups: []string{"coordination.k8s.io"},
			Resources: []string{"leases"},
		},
		{
			Verbs: []string{
				"create",
				"patch",
			},
			APIGroups: []string{""},
			Resources: []string{"events"},
		},
	}
	assert.Equal(t, rules, rbac.Rules)
}
