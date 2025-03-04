// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package unit

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestRBACRoleBindingCreateTrue(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "true",
		},
		Version: helmChartVersion,
	}
	rbac := renderRBACRoleBindingManifest(t, options)

	assert.Equal(t, defaultRBACRoleBindingName, rbac.Name)
	assert.Empty(t, rbac.Labels)
	assert.Empty(t, rbac.Annotations)
	assert.Equal(t, defaultNamespace, rbac.Namespace)

	testRBACRoleBindingRoleRef(t, rbac)
	testRBACRoleBindingSubjects(t, rbac, defaultNamespace)
}

func TestRBACRoleBindingCreateFalse(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "false",
		},
		Version: helmChartVersion,
	}
	_, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacRoleBindingTemplate})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find template")
}

func TestRBACRoleBindingNamespace(t *testing.T) {
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
	rbac := renderRBACRoleBindingManifest(t, options)

	assert.Equal(t, defaultRBACRoleBindingName, rbac.Name)
	assert.Empty(t, rbac.Labels)
	assert.Empty(t, rbac.Annotations)
	assert.Equal(t, ns, rbac.Namespace)

	testRBACRoleBindingRoleRef(t, rbac)
	testRBACRoleBindingSubjects(t, rbac, ns)
}

func testRBACRoleBindingRoleRef(t *testing.T, rbac rbacv1.RoleBinding) {
	roleRef := rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "Role",
		Name:     defaultRBACRoleName,
	}
	assert.Equal(t, roleRef, rbac.RoleRef)
}

func testRBACRoleBindingSubjects(t *testing.T, rbac rbacv1.RoleBinding, ns string) {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      defaultServiceAccountName,
			Namespace: ns,
		},
	}
	assert.Equal(t, subjects, rbac.Subjects)
}
