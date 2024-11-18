// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package test

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	helmChartName    = "hcp-terraform-operator"
	helmChartVersion = "2.7.0"
	helmChartPath    = "../hcp-terraform-operator"
	helmReleaseName  = "this"

	defaultNamespace = "default"

	serviceAccountTeamplate = "templates/serviceaccount.yaml"
)

var (
	defaultServiceAccountLabels = map[string]string{
		"helm.sh/chart":                fmt.Sprintf("%s-%s", helmChartName, helmChartVersion),
		"app.kubernetes.io/name":       helmChartName,
		"app.kubernetes.io/instance":   helmReleaseName,
		"app.kubernetes.io/version":    helmChartVersion,
		"app.kubernetes.io/managed-by": "Helm",
	}

	defaultServiceAccountName = fmt.Sprintf("%s-%s", helmReleaseName, helmChartName)
)

func TestControllerServiceAccountCreateTrue(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "true",
		},
		Version: helmChartVersion,
	}

	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{serviceAccountTeamplate})
	assert.NoError(t, err)

	var sa corev1.ServiceAccount
	helm.UnmarshalK8SYaml(t, output, &sa)

	assert.Equal(t, defaultServiceAccountName, sa.Name)
	assert.Equal(t, defaultServiceAccountLabels, sa.Labels)
	assert.Empty(t, sa.Annotations)
	assert.Equal(t, defaultNamespace, sa.Namespace)
}

func TestControllerServiceAccountCreateFalse(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "false",
		},
		Version: helmChartVersion,
	}

	_, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{"templates/serviceaccount.yaml"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("could not find template %s in chart", serviceAccountTeamplate))
}

func TestControllerServiceAccountAnnotations(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "true",
		},
		SetJsonValues: map[string]string{
			"serviceAccount.annotations": `{"app.kubernetes.io/name": "hcp-terraform-operator"}`,
		},
		Version: helmChartVersion,
	}

	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{serviceAccountTeamplate})
	assert.NoError(t, err)

	var sa corev1.ServiceAccount
	helm.UnmarshalK8SYaml(t, output, &sa)

	expectedAnnotations := map[string]string{
		"app.kubernetes.io/name": "hcp-terraform-operator",
	}

	assert.Equal(t, defaultServiceAccountName, sa.Name)
	assert.Equal(t, defaultServiceAccountLabels, sa.Labels)
	assert.Equal(t, expectedAnnotations, sa.Annotations)
	assert.Equal(t, defaultNamespace, sa.Namespace)
}

func TestControllerServiceAccountNamespace(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "true",
		},
		Version: helmChartVersion,
	}

	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{serviceAccountTeamplate})
	assert.NoError(t, err)

	var sa corev1.ServiceAccount
	helm.UnmarshalK8SYaml(t, output, &sa)

	assert.Equal(t, defaultServiceAccountName, sa.Name)
	assert.Equal(t, defaultServiceAccountLabels, sa.Labels)
	assert.Empty(t, sa.Annotations)
	assert.Equal(t, defaultNamespace, sa.Namespace)
}

func TestControllerServiceAccountName(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "true",
			"serviceAccount.name":   "this",
		},
		Version: helmChartVersion,
	}

	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{serviceAccountTeamplate})
	assert.NoError(t, err)

	var sa corev1.ServiceAccount
	helm.UnmarshalK8SYaml(t, output, &sa)

	expectedName := "this"

	assert.Equal(t, expectedName, sa.Name)
	assert.Equal(t, defaultServiceAccountLabels, sa.Labels)
	assert.Empty(t, sa.Annotations)
	assert.Equal(t, defaultNamespace, sa.Namespace)
}
