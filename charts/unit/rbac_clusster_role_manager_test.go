// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestRBACClusterRoleManagerCreateTrue(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "true",
		},
		Version: helmChartVersion,
	}
	rbac := renderRBACClusterRoleManagerManifest(t, options)

	assert.Equal(t, defaultRBACClusterRoleManagerName, rbac.Name)
	assert.Empty(t, rbac.Labels)
	assert.Empty(t, rbac.Annotations)

	testRBACClusterRoleManagerRules(t, rbac)
}

func TestRBAClusterRoleManagerCreateFalse(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "false",
		},
		Version: helmChartVersion,
	}
	_, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacClusterRoleManagerTemplate})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find template")
}

func testRBACClusterRoleManagerRules(t *testing.T, rbac rbacv1.ClusterRole) {
	rules := []rbacv1.PolicyRule{
		{
			Verbs: []string{
				"create",
				"list",
				"update",
				"watch",
			},
			APIGroups: []string{""},
			Resources: []string{"configmaps"},
		},
		{
			Verbs: []string{
				"create",
				"patch",
			},
			APIGroups: []string{""},
			Resources: []string{"events"},
		},
		{
			Verbs: []string{
				"create",
				"list",
				"update",
				"watch",
			},
			APIGroups: []string{""},
			Resources: []string{"secrets"},
		},
		{
			Verbs: []string{
				"create",
				"delete",
				"get",
				"list",
				"patch",
				"update",
				"watch",
			},
			APIGroups: []string{"apps"},
			Resources: []string{"deployments"},
		},
		{
			Verbs: []string{
				"create",
				"delete",
				"get",
				"list",
				"patch",
				"update",
				"watch",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"agentpools"},
		},
		{
			Verbs: []string{
				"update",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"agentpools/finalizers"},
		},
		{
			Verbs: []string{
				"get",
				"patch",
				"update",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"agentpools/status"},
		},
		{
			Verbs: []string{
				"create",
				"delete",
				"get",
				"list",
				"patch",
				"update",
				"watch",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"modules"},
		},
		{
			Verbs: []string{
				"update",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"modules/finalizers"},
		},
		{
			Verbs: []string{
				"get",
				"patch",
				"update",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"modules/status"},
		},
		{
			Verbs: []string{
				"create",
				"delete",
				"get",
				"list",
				"patch",
				"update",
				"watch",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"projects"},
		},
		{
			Verbs: []string{
				"update",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"projects/finalizers"},
		},
		{
			Verbs: []string{
				"get",
				"patch",
				"update",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"projects/status"},
		},
		{
			Verbs: []string{
				"create",
				"delete",
				"get",
				"list",
				"patch",
				"update",
				"watch",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"workspaces"},
		},
		{
			Verbs: []string{
				"update",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"workspaces/finalizers"},
		},
		{
			Verbs: []string{
				"get",
				"patch",
				"update",
			},
			APIGroups: []string{"app.terraform.io"},
			Resources: []string{"workspaces/status"},
		},
	}
	assert.Equal(t, rules, rbac.Rules)
}
