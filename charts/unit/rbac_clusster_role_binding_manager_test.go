// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
)

func TestRBACClusterRoleBindingManagerCreateTrue(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"rbac.create": "true",
		},
		Version: helmChartVersion,
	}
	rbac := renderRBACClusterRoleBindingManagerManifest(t, options)

	assert.Equal(t, defaultRBACClusterRoleBindingManagerName, rbac.Name)
	assert.Empty(t, rbac.Labels)
	assert.Empty(t, rbac.Annotations)

	testRBACClusterRoleBindingManagerRoleRef(t, rbac)
	testRBACClusterRoleBindingManagerSubjects(t, rbac, defaultNamespace)
}

func TestRBACClusterRoleBindingManagerCreateFalse(t *testing.T) {
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

func testRBACClusterRoleBindingManagerRoleRef(t *testing.T, rbac rbacv1.ClusterRoleBinding) {
	roleRef := rbacv1.RoleRef{
		APIGroup: rbacv1.GroupName,
		Kind:     "ClusterRole",
		Name:     defaultRBACClusterRoleManagerName,
	}
	assert.Equal(t, roleRef, rbac.RoleRef)
}

func testRBACClusterRoleBindingManagerSubjects(t *testing.T, rbac rbacv1.ClusterRoleBinding, ns string) {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      defaultServiceAccountName,
			Namespace: ns,
		},
	}
	assert.Equal(t, subjects, rbac.Subjects)
}
