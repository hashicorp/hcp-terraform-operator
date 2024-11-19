// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package unit

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	spec := deployment.Spec.Template.Spec
	// Template.Spec
	assert.Empty(t, spec.PriorityClassName)
	assert.Empty(t, spec.ImagePullSecrets)
	assert.Len(t, spec.Containers, 2)
	assert.Equal(t, corev1.Container{
		Name:    "manager",
		Image:   "hashicorp/hcp-terraform-operator:2.7.0",
		Command: []string{"/manager"},
		Args: []string{
			"--sync-period=5m",
			"--agent-pool-workers=1",
			"--agent-pool-sync-period=30s",
			"--module-workers=1",
			"--module-sync-period=5m",
			"--project-workers=1",
			"--project-sync-period=5m",
			"--workspace-workers=1",
			"--workspace-sync-period=5m",
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8081,
					},
				},
			},
			InitialDelaySeconds: 15,
			PeriodSeconds:       20,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: 8081,
					},
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			AllowPrivilegeEscalation: ptr.To(false),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
	}, spec.Containers[0])
	assert.Equal(t, corev1.Container{
		Name:  "kube-rbac-proxy",
		Image: "quay.io/brancz/kube-rbac-proxy:v0.18.0",
		Args: []string{
			"--secure-listen-address=0.0.0.0:8443",
			"--upstream=http://127.0.0.1:8080/",
			"--logtostderr=true",
			"--v=0",
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "https",
				ContainerPort: 8443,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			AllowPrivilegeEscalation: ptr.To(false),
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
	}, spec.Containers[1])

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
