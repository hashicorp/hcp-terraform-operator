// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package unit

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	// Helm chart.
	helmChartName   = "hcp-terraform-operator"
	helmChartPath   = "../../hcp-terraform-operator"
	helmReleaseName = "this"

	// Defaults.
	defaultNamespace = "default"

	// Templates.
	deploymentTemplate                    = "templates/deployment.yaml"
	rbacRoleTemplate                      = "templates/role.yaml"
	rbacRoleBindingTemplate               = "templates/rolebinding.yaml"
	rbacClusterRoleManagerTemplate        = "templates/clusterrole_manager.yaml"
	rbacClusterRoleMetricsReaderTemplate  = "templates/clusterrole_metrics_reader.yaml"
	rbacClusterRoleProxyTemplate          = "templates/clusterrole_proxy.yaml"
	rbacClusterRoleBindingManagerTemplate = "templates/clusterrolebinding_manager.yaml"
	rbacClusterRoleBindingProxyTemplate   = "templates/clusterrolebinding_proxy.yaml"
	serviceAccountTemplate                = "templates/serviceaccount.yaml"
)

var (
	// Generic variables.
	helmChartVersion = "0.0.0"

	// Deployment variables.
	defaultDeploymentName   = fmt.Sprintf("%s-%s", helmReleaseName, helmChartName)
	defaultDeploymentLabels = map[string]string{
		"helm.sh/chart":                fmt.Sprintf("%s-%s", helmChartName, helmChartVersion),
		"app.kubernetes.io/name":       helmChartName,
		"app.kubernetes.io/instance":   helmReleaseName,
		"app.kubernetes.io/version":    helmChartVersion,
		"control-plane":                fmt.Sprintf("%s-controller-manager", helmReleaseName),
		"app.kubernetes.io/managed-by": "Helm",
	}
	defaultDeploymentReplicas       = int32(2)
	defaultDeploymentSelectorLabels = map[string]string{
		"app.kubernetes.io/instance": helmReleaseName,
		"app.kubernetes.io/name":     helmChartName,
		"control-plane":              fmt.Sprintf("%s-controller-manager", helmReleaseName),
	}
	defaultDeploymentTemplateSpecLabels = map[string]string{
		"control-plane": fmt.Sprintf("%s-controller-manager", helmReleaseName),
	}
	defaultDeploymentTerminationGracePeriodSeconds = int64(10)
	defaultDeploymentTemplateVolumeName            = "manager-config"
	defaultDeploymentTemplateVolumeConfigMapName   = fmt.Sprintf("%s-manager-config", helmReleaseName)

	// RBAC variables.
	defaultRBACRoleName        = fmt.Sprintf("%s-leader-election-role", helmReleaseName)
	defaultRBACRoleBindingName = fmt.Sprintf("%s-leader-election-rolebinding", helmReleaseName)

	defaultRBACClusterRoleManagerName        = fmt.Sprintf("%s-manager-role", helmReleaseName)
	defaultRBACClusterRoleMetricsReaderName  = fmt.Sprintf("%s-metrics-reader", helmReleaseName)
	defaultRBACClusterRoleProxyName          = fmt.Sprintf("%s-proxy-role", helmReleaseName)
	defaultRBACClusterRoleBindingManagerName = fmt.Sprintf("%s-manager-rolebinding", helmReleaseName)
	defaultRBACClusterRoleBindingProxyName   = fmt.Sprintf("%s-proxy-rolebinding", helmReleaseName)

	// Service account variables.
	defaultServiceAccountName   = fmt.Sprintf("%s-%s", helmReleaseName, helmChartName)
	defaultServiceAccountLabels = map[string]string{
		"helm.sh/chart":                fmt.Sprintf("%s-%s", helmChartName, helmChartVersion),
		"app.kubernetes.io/name":       helmChartName,
		"app.kubernetes.io/instance":   helmReleaseName,
		"app.kubernetes.io/version":    helmChartVersion,
		"app.kubernetes.io/managed-by": "Helm",
	}
)

func init() {
	var err error
	if helmChartVersion, err = getChartVersion(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	defaultDeploymentLabels["helm.sh/chart"] = fmt.Sprintf("%s-%s", helmChartName, helmChartVersion)
	defaultDeploymentLabels["app.kubernetes.io/version"] = helmChartVersion

	defaultServiceAccountLabels["helm.sh/chart"] = fmt.Sprintf("%s-%s", helmChartName, helmChartVersion)
	defaultServiceAccountLabels["app.kubernetes.io/version"] = helmChartVersion
}

type Chart struct {
	Version string `yaml:"version"`
}

func getChartVersion() (string, error) {
	file, err := os.ReadFile(fmt.Sprintf("%s/Chart.yaml", helmChartPath))
	if err != nil {
		log.Fatalf("Error reading Chart.yaml: %v", err)
	}

	var chart Chart
	if err := yaml.Unmarshal(file, &chart); err != nil {
		return "", fmt.Errorf("Error unmarshalling YAML: %v", err)
	}

	return chart.Version, nil
}

func renderDeploymentManifest(t *testing.T, options *helm.Options) appsv1.Deployment {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{deploymentTemplate})
	assert.NoError(t, err)

	deployment := appsv1.Deployment{}
	helm.UnmarshalK8SYaml(t, output, &deployment)

	return deployment
}

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

func renderRBACClusterRoleManagerManifest(t *testing.T, options *helm.Options) rbacv1.ClusterRole {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacClusterRoleManagerTemplate})
	assert.NoError(t, err)

	rbac := rbacv1.ClusterRole{}
	helm.UnmarshalK8SYaml(t, output, &rbac)

	return rbac
}

func renderRBACClusterRoleMetricsReaderManifest(t *testing.T, options *helm.Options) rbacv1.ClusterRole {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacClusterRoleMetricsReaderTemplate})
	assert.NoError(t, err)

	rbac := rbacv1.ClusterRole{}
	helm.UnmarshalK8SYaml(t, output, &rbac)

	return rbac
}

func renderRBACClusterRoleProxyManifest(t *testing.T, options *helm.Options) rbacv1.ClusterRole {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacClusterRoleProxyTemplate})
	assert.NoError(t, err)

	rbac := rbacv1.ClusterRole{}
	helm.UnmarshalK8SYaml(t, output, &rbac)

	return rbac
}

func renderRBACClusterRoleBindingManagerManifest(t *testing.T, options *helm.Options) rbacv1.ClusterRoleBinding {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacClusterRoleBindingManagerTemplate})
	assert.NoError(t, err)

	rbac := rbacv1.ClusterRoleBinding{}
	helm.UnmarshalK8SYaml(t, output, &rbac)

	return rbac
}

func renderRBACClusterRoleBindingProxyManifest(t *testing.T, options *helm.Options) rbacv1.ClusterRoleBinding {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{rbacClusterRoleBindingProxyTemplate})
	assert.NoError(t, err)

	rbac := rbacv1.ClusterRoleBinding{}
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
