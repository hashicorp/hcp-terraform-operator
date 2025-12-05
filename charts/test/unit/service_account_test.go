// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package unit

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
)

func TestServiceAccountCreateTrue(t *testing.T) {
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

func TestServiceAccountCreateFalse(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "false",
		},
		Version: helmChartVersion,
	}
	_, err := helm.RenderTemplateE(t, options, helmChartPath, helmReleaseName, []string{serviceAccountTemplate})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not find template")
}

func TestServiceAccountAnnotations(t *testing.T) {
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

func TestServiceAccountNamespace(t *testing.T) {
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

func TestServiceAccountName(t *testing.T) {
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
