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

	serviceAccountTemplate = "templates/serviceaccount.yaml"
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
)

func renderServiceAccountManifest(t *testing.T, options *helm.Options) corev1.ServiceAccount {
	output, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{serviceAccountTemplate})
	assert.NoError(t, err)

	sa := corev1.ServiceAccount{}
	helm.UnmarshalK8SYaml(t, output, &sa)
	return sa
}
