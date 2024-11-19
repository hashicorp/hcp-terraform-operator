// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestDeploymentDefault(t *testing.T) {
	options := &helm.Options{
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)

	assert.Equal(t, defaultDeploymentName, deployment.Name)
	assert.Equal(t, defaultDeploymentLabels, deployment.Labels)
	assert.Empty(t, deployment.Annotations)
	assert.Equal(t, defaultNamespace, deployment.Namespace)

	assert.Equal(t, &defaultDeploymentReplicas, deployment.Spec.Replicas)
	assert.Equal(t, &metav1.LabelSelector{MatchLabels: defaultDeploymentSelectorLabels}, deployment.Spec.Selector)

	assert.Empty(t, deployment.Spec.Template.Annotations)
	assert.Equal(t, defaultDeploymentSelectorLabels, deployment.Spec.Template.Labels)

	// Template.Spec

	assert.Equal(t, defaultServiceAccountName, deployment.Spec.Template.Spec.ServiceAccountName)
	assert.Equal(t, &corev1.PodSecurityContext{RunAsNonRoot: ptr.To(true)}, deployment.Spec.Template.Spec.SecurityContext)
	assert.Equal(t, &defaultDeploymentTerminationGracePeriodSeconds, deployment.Spec.Template.Spec.TerminationGracePeriodSeconds)
	assert.Equal(t, []corev1.Volume{
		{
			Name: defaultDeploymentTemplateVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: defaultDeploymentTemplateVolumeConfigMapName,
					},
				},
			},
		},
	}, deployment.Spec.Template.Spec.Volumes)
}
