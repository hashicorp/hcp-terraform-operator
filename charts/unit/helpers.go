// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package test

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	// Helm chart.
	helmChartName    = "hcp-terraform-operator"
	helmChartVersion = "2.7.0"
	helmChartPath    = "../hcp-terraform-operator"
	helmReleaseName  = "this"

	// Defaults.
	defaultNamespace = "default"

	// Templates.
	deploymentTemplate             = "templates/deployment.yaml"
	rbacRoleTemplate               = "templates/role.yaml"
	rbacRoleBindingTemplate        = "templates/rolebinding.yaml"
	rbacClusterRoleTemplate        = "templates/clusterrole.yaml"
	rbacClusterRoleBindingTemplate = "templates/clusterrolebinding.yaml"
	serviceAccountTemplate         = "templates/serviceaccount.yaml"
)

var (
	defaultServiceAccountName   = fmt.Sprintf("%s-%s", helmReleaseName, helmChartName)
	defaultServiceAccountLabels = map[string]string{
		"helm.sh/chart":                fmt.Sprintf("%s-%s", helmChartName, helmChartVersion),
		"app.kubernetes.io/name":       helmChartName,
		"app.kubernetes.io/instance":   helmReleaseName,
		"app.kubernetes.io/version":    helmChartVersion,
		"app.kubernetes.io/managed-by": "Helm",
	}

	defaultRBACRoleName        = fmt.Sprintf("%s-leader-election-role", helmReleaseName)
	defaultRBACRoleBindingName = fmt.Sprintf("%s-leader-election-rolebinding", helmReleaseName)
)

func renderRBACRoleManifest(t *testing.T, options *helm.Options) rbacv1.Role {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacRoleTemplate})
	assert.NoError(t, err)

	rbac := rbacv1.Role{}
	helm.UnmarshalK8SYaml(t, output, &rbac)

	return rbac
}

func renderRBACRoleBindingManifest(t *testing.T, options *helm.Options) rbacv1.RoleBinding {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacRoleBindingTemplate})
	assert.NoError(t, err)

	rbac := rbacv1.RoleBinding{}
	helm.UnmarshalK8SYaml(t, output, &rbac)

	return rbac
}

func renderServiceAccountManifest(t *testing.T, options *helm.Options) corev1.ServiceAccount {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{serviceAccountTemplate})
	assert.NoError(t, err)

	sa := corev1.ServiceAccount{}
	helm.UnmarshalK8SYaml(t, output, &sa)

	return sa
}
