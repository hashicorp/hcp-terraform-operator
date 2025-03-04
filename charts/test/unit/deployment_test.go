// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package unit

import (
	"fmt"
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
							Image:   fmt.Sprintf("hashicorp/hcp-terraform-operator:%s", helmChartVersion),
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
							Image: fmt.Sprintf("%s:%s", helmChartValues.KubeRbacProxy.Image.Repository, helmChartValues.KubeRbacProxy.Image.Tag),
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
	dd := defaultDeployment()

	assert.Equal(t, dd, deployment)
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
	dd := defaultDeployment()
	dd.Namespace = ns

	assert.Equal(t, dd, deployment)
}

func TestDeploymentImagePullSecrets(t *testing.T) {
	options := &helm.Options{
		SetJsonValues: map[string]string{
			"imagePullSecrets": `[{"name": "this"}]`,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
		{
			Name: "this",
		},
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentPodLabels(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"podLabels.this": "this",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	labels := dd.Spec.Template.DeepCopy().Labels
	labels["this"] = "this"
	dd.Spec.Template.Labels = labels

	assert.Equal(t, dd, deployment)
}

func TestDeploymentReplicaCount(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"replicaCount": "5",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Replicas = ptr.To(int32(5))

	assert.Equal(t, dd, deployment)
}

func TestDeploymentSecurityContext(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"securityContext.runAsNonRoot": "false",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.SecurityContext.RunAsNonRoot = ptr.To(false)

	assert.Equal(t, dd, deployment)
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
	dd := defaultDeployment()
	dd.Spec.Template.Spec.PriorityClassName = priorityClassName

	assert.Equal(t, dd, deployment)
}

func TestDeploymentOperatorImage(t *testing.T) {
	options := &helm.Options{
		Version: helmChartVersion,
		SetValues: map[string]string{
			"operator.image.repository": "this",
			"operator.image.pullPolicy": string(corev1.PullAlways),
			"operator.image.tag":        "0.0.1",
		},
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[0].Image = "this:0.0.1"
	dd.Spec.Template.Spec.Containers[0].ImagePullPolicy = corev1.PullAlways

	assert.Equal(t, dd, deployment)
}

func TestDeploymentOperatorResources(t *testing.T) {
	options := &helm.Options{
		SetJsonValues: map[string]string{
			"operator.resources.limits": `{
				"cpu": "2",
				"memory": "512Mi"
			}`,
			"operator.resources.requests": `{
				"cpu": "250m",
				"memory": "128Mi"
			}`,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentOperatorSecurityContext(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"operator.securityContext.allowPrivilegeEscalation": "true",
			"operator.securityContext.runAsNonRoot":             "true",
			"operator.securityContext.seccompProfile.type":      "Localhost",
			"operator.securityContext.capabilities.add":         `{NET_BIND_SERVICE}`,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[0].SecurityContext = &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(true),
		RunAsNonRoot:             ptr.To(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				corev1.Capability("ALL"),
			},
			Add: []corev1.Capability{
				corev1.Capability("NET_BIND_SERVICE"),
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeLocalhost,
		},
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentOperatorAffinity(t *testing.T) {
	options := &helm.Options{
		SetJsonValues: map[string]string{
			"operator.affinity": `{
				"nodeAffinity": {
					"requiredDuringSchedulingIgnoredDuringExecution": {
						"nodeSelectorTerms": [{
							"matchExpressions": [{
								"key": "kubernetes.io/arch",
								"operator": "In",
								"values": ["amd64"]
							}]
						}]
					}
				}
			}`,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Affinity = &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/arch",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{"amd64"},
							},
						},
					},
				},
			},
		},
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentOperatorTolerations(t *testing.T) {
	options := &helm.Options{
		SetJsonValues: map[string]string{
			"operator.tolerations": `[{
				"key": "cloud.hashicorp.com/terraform",
				"operator": "Equal",
				"value": "operator",
				"effect": "PreferNoSchedule"
			}]`,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Tolerations = []corev1.Toleration{
		{
			Key:      "cloud.hashicorp.com/terraform",
			Operator: corev1.TolerationOpEqual,
			Value:    "operator",
			Effect:   corev1.TaintEffectPreferNoSchedule,
		},
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentOperatorSyncPeriod(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"operator.syncPeriod": "4h",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[0].Args = []string{
		"--sync-period=4h",
		"--agent-pool-workers=1",
		"--agent-pool-sync-period=30s",
		"--module-workers=1",
		"--module-sync-period=5m",
		"--project-workers=1",
		"--project-sync-period=5m",
		"--workspace-workers=1",
		"--workspace-sync-period=5m",
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentOperatorWatchedNamespaces(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"operator.watchedNamespaces": `{white,blue,red}`,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[0].Args = append(dd.Spec.Template.Spec.Containers[0].Args, []string{
		"--namespace=white",
		"--namespace=blue",
		"--namespace=red",
	}...)

	assert.Equal(t, dd, deployment)
}

func TestDeploymentOperatorTFEAddress(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"operator.tfeAddress": "https://tfe.hashi.co",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
		{
			Name:  "TFE_ADDRESS",
			Value: "https://tfe.hashi.co",
		},
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentOperatorSkipTLSVerify(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"operator.skipTLSVerify": "true",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[0].Env = []corev1.EnvVar{
		{
			Name:  "TFC_TLS_SKIP_VERIFY",
			Value: "true",
		},
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentKubeRbacProxyImage(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"kubeRbacProxy.image.repository": "this",
			"kubeRbacProxy.image.pullPolicy": string(corev1.PullAlways),
			"kubeRbacProxy.image.tag":        "0.0.1",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[1].Image = "this:0.0.1"
	dd.Spec.Template.Spec.Containers[1].ImagePullPolicy = corev1.PullAlways

	assert.Equal(t, dd, deployment)
}

func TestDeploymentKubeRbacProxySecurityContext(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"kubeRbacProxy.securityContext.allowPrivilegeEscalation": "true",
			"kubeRbacProxy.securityContext.runAsNonRoot":             "true",
			"kubeRbacProxy.securityContext.seccompProfile.type":      "Localhost",
			"kubeRbacProxy.securityContext.capabilities.add":         `{NET_BIND_SERVICE}`,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[1].SecurityContext = &corev1.SecurityContext{
		AllowPrivilegeEscalation: ptr.To(true),
		RunAsNonRoot:             ptr.To(true),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				corev1.Capability("ALL"),
			},
			Add: []corev1.Capability{
				corev1.Capability("NET_BIND_SERVICE"),
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeLocalhost,
		},
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentKubeRbacProxyResources(t *testing.T) {
	options := &helm.Options{
		SetJsonValues: map[string]string{
			"kubeRbacProxy.resources.limits": `{
				"cpu": "2",
				"memory": "512Mi"
			}`,
			"kubeRbacProxy.resources.requests": `{
				"cpu": "250m",
				"memory": "128Mi"
			}`,
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[1].Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("2"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("128Mi"),
		},
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentControllers(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"controllers.agentPool.workers":    "5",
			"controllers.agentPool.syncPeriod": "15m",
			"controllers.module.workers":       "5",
			"controllers.module.syncPeriod":    "15m",
			"controllers.project.workers":      "5",
			"controllers.project.syncPeriod":   "15m",
			"controllers.workspace.workers":    "5",
			"controllers.workspace.syncPeriod": "15m",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Containers[0].Args = []string{
		"--sync-period=1h",
		"--agent-pool-workers=5",
		"--agent-pool-sync-period=15m",
		"--module-workers=5",
		"--module-sync-period=15m",
		"--project-workers=5",
		"--project-sync-period=15m",
		"--workspace-workers=5",
		"--workspace-sync-period=15m",
	}

	assert.Equal(t, dd, deployment)
}

func TestDeploymentCustomCAcertificates(t *testing.T) {
	options := &helm.Options{
		SetValues: map[string]string{
			"customCAcertificates": "SGVsbG8gV29ybGQ=",
		},
		Version: helmChartVersion,
	}
	deployment := renderDeploymentManifest(t, options)
	dd := defaultDeployment()
	dd.Spec.Template.Spec.Volumes = []corev1.Volume{
		{
			Name: "ca-certificates",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-ca-certificates", helmReleaseName),
					},
				},
			},
		},
	}
	dd.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "ca-certificates",
			ReadOnly:  true,
			MountPath: "/etc/ssl/certs/custom-ca-certificates.crt",
			SubPath:   "ca-certificates",
		},
	}

	assert.Equal(t, dd, deployment)
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
	dd := defaultDeployment()
	dd.Spec.Template.Spec.ServiceAccountName = serviceAccountName

	assert.Equal(t, dd, deployment)
}
