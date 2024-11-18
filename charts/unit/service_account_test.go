// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package test

import (
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
)

func TestControllerServiceAccountCreateTrue(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "true",
		},
		Version: helmChartVersion,
	}
	sa := renderServiceAccountManifest(t, options)

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
	assert.Contains(t, err.Error(), fmt.Sprintf("could not find template %s in chart", serviceAccountTemplate))
}

func TestControllerServiceAccountAnnotations(t *testing.T) {
	expectedAnnotations := map[string]string{
		"app.kubernetes.io/name": "hcp-terraform-operator",
	}
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "true",
		},
		SetJsonValues: map[string]string{
			"serviceAccount.annotations": `{"app.kubernetes.io/name": "hcp-terraform-operator"}`,
		},
		Version: helmChartVersion,
	}
	sa := renderServiceAccountManifest(t, options)

	assert.Equal(t, defaultServiceAccountName, sa.Name)
	assert.Equal(t, defaultServiceAccountLabels, sa.Labels)
	assert.Equal(t, expectedAnnotations, sa.Annotations)
	assert.Equal(t, defaultNamespace, sa.Namespace)
}

// FINISH
func TestControllerServiceAccountNamespace(t *testing.T) {
	ns := "this"
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "true",
		},
		EnvVars: map[string]string{
			"HELM_NAMESPACE": ns,
		},
		Version: helmChartVersion,
	}
	sa := renderServiceAccountManifest(t, options)

	assert.Equal(t, defaultServiceAccountName, sa.Name)
	assert.Equal(t, defaultServiceAccountLabels, sa.Labels)
	assert.Empty(t, sa.Annotations)
	assert.Equal(t, ns, sa.Namespace)
}

func TestControllerServiceAccountName(t *testing.T) {
	name := "this"
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "true",
			"serviceAccount.name":   name,
		},
		Version: helmChartVersion,
	}
	sa := renderServiceAccountManifest(t, options)

	assert.Equal(t, name, sa.Name)
	assert.Equal(t, defaultServiceAccountLabels, sa.Labels)
	assert.Empty(t, sa.Annotations)
	assert.Equal(t, defaultNamespace, sa.Namespace)
}
