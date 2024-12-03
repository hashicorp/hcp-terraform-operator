// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package unit

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func defaultDeployment() appsv1.Deployment {
	return appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultDeploymentName,
			Namespace: defaultNamespace,
			Labels:    defaultDeploymentLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(defaultDeploymentReplicas),
			Selector: &metav1.LabelSelector{MatchLabels: defaultDeploymentSelectorLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultDeploymentSelectorLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            defaultServiceAccountName,
					SecurityContext:               &corev1.PodSecurityContext{RunAsNonRoot: ptr.To(true)},
					TerminationGracePeriodSeconds: &defaultDeploymentTerminationGracePeriodSeconds,
					Containers: []corev1.Container{
						{
							Name:    "manager",
							Image:   "hashicorp/hcp-terraform-operator:2.7.0",
							Command: []string{"/manager"},
							Args: []string{
								"--sync-period=1h",
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
						},
						{
							Name:  "kube-rbac-proxy",
							Image: "quay.io/brancz/kube-rbac-proxy:v0.18.2",
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
						},
					},
				},
			},
		},
	}
}

func TestDeploymentDefault(t *testing.T) {
	options := &helm.Options{
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	d := defaultDeployment()

	assert.Equal(t, d, deployment)
}

func TestDeploymentNamespace(t *testing.T) {
	ns := "this"
	options := &helm.Options{
		EnvVars: map[string]string{
			"HELM_NAMESPACE": ns,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	d := defaultDeployment()
	d.Namespace = ns

	assert.Equal(t, d, deployment)
}

func TestDeploymentReplicas(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"replicaCount": "5",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	d := defaultDeployment()
	d.Spec.Replicas = ptr.To(int32(5))

	assert.Equal(t, d, deployment)
}

func TestDeploymentPriorityClassName(t *testing.T) {
	priorityClassName := "high-priority"
	options := &helm.Options{
		SetValues: map[string]string{
			"priorityClassName": priorityClassName,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	d := defaultDeployment()
	d.Spec.Template.Spec.PriorityClassName = priorityClassName

	assert.Equal(t, d, deployment)
}

func TestDeploymentServiceAccountName(t *testing.T) {
	serviceAccountName := "this"
	options := &helm.Options{
		SetValues: map[string]string{
			"serviceAccount.create": "true",
			"serviceAccount.name":   serviceAccountName,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	d := defaultDeployment()
	d.Spec.Template.Spec.ServiceAccountName = serviceAccountName

	assert.Equal(t, d, deployment)
}

func TestDeploymentImagePullSecrets(t *testing.T) {
	options := &helm.Options{
		SetJsonValues: map[string]string{
			"imagePullSecrets": `[{"name": "this"}]`,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	d := defaultDeployment()
	d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: "this",
		},
	}

	assert.Equal(t, d, deployment)
}

func TestDeploymentPodLabels(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"podLabels.this": "this",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	d := defaultDeployment()

	labels := d.Spec.Template.DeepCopy().Labels
	labels["this"] = "this"
	d.Spec.Template.Labels = labels

	assert.Equal(t, d, deployment)
}

// TODO:
// - customCAcertificates
// - securityContext
// - kubeRbacProxy.*
// - operator.*
// - controllers.*
