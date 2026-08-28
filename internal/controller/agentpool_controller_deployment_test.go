// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	tfc "github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func newTestTFCServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/ping" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func newTestAgentPoolInstance(t *testing.T, name, namespace string) *agentPoolInstance {
	t.Helper()

	tfServer := newTestTFCServer()
	t.Cleanup(tfServer.Close)

	client, err := tfc.NewClient(&tfc.Config{
		Address: tfServer.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)

	return &agentPoolInstance{
		instance: appv1alpha2.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				UID:       types.UID("test-uid"),
			},
			Spec: appv1alpha2.AgentPoolSpec{
				Name:         name,
				Organization: "test-org",
				AgentDeployment: &appv1alpha2.AgentDeployment{
					Replicas: pointer.PointerOf(int32(3)),
				},
			},
			Status: appv1alpha2.AgentPoolStatus{
				AgentPoolID: "apool-123",
				AgentTokens: []*appv1alpha2.AgentAPIToken{
					{Name: "test-token"},
				},
			},
		},
		log:      logr.Discard(),
		tfClient: HCPTerraformClient{Client: client},
	}
}

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = appv1alpha2.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	return s
}

func TestDecorateDeployment_MixedContainerEnvSets(t *testing.T) {
	t.Parallel()

	tfServer := newTestTFCServer()
	defer tfServer.Close()

	client, err := tfc.NewClient(&tfc.Config{
		Address: tfServer.URL,
		Token:   "test-token",
	})
	require.NoError(t, err)

	ap := &agentPoolInstance{
		instance: appv1alpha2.AgentPool{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pool-a",
				Namespace: "default",
			},
			Status: appv1alpha2.AgentPoolStatus{
				AgentTokens: []*appv1alpha2.AgentAPIToken{{Name: "token-a"}},
			},
		},
		tfClient: HCPTerraformClient{Client: client},
	}

	d := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "agent-0",
							Env: []corev1.EnvVar{
								{Name: "TFC_AGENT_AUTO_UPDATE", Value: "minor"},
								{Name: "TFC_ADDRESS", Value: "https://already-set.example.com"},
							},
						},
						{
							Name: "agent-1",
							Env: []corev1.EnvVar{
								{Name: "TFC_AGENT_NAME", Value: "custom-agent-name"},
							},
						},
					},
				},
			},
		},
	}

	decorateDeployment(ap, d)

	container0 := d.Spec.Template.Spec.Containers[0]
	assert.Equal(t, 1, countEnvVar(container0.Env, "TFC_AGENT_AUTO_UPDATE"))
	assert.Equal(t, "minor", mustFindEnvVar(t, container0.Env, "TFC_AGENT_AUTO_UPDATE").Value)
	assert.Equal(t, 1, countEnvVar(container0.Env, "TFC_AGENT_TOKEN"))
	assert.Equal(
		t,
		agentPoolOutputObjectName(ap.instance.Name),
		mustFindEnvVar(t, container0.Env, "TFC_AGENT_TOKEN").ValueFrom.SecretKeyRef.Name,
	)
	assert.Equal(t, "token-a", mustFindEnvVar(t, container0.Env, "TFC_AGENT_TOKEN").ValueFrom.SecretKeyRef.Key)
	assert.Equal(t, 1, countEnvVar(container0.Env, "TFC_AGENT_NAME"))
	assert.Equal(t, "metadata.name", mustFindEnvVar(t, container0.Env, "TFC_AGENT_NAME").ValueFrom.FieldRef.FieldPath)
	assert.Equal(t, 1, countEnvVar(container0.Env, "TFC_ADDRESS"))
	assert.Equal(t, "https://already-set.example.com", mustFindEnvVar(t, container0.Env, "TFC_ADDRESS").Value)

	container1 := d.Spec.Template.Spec.Containers[1]
	customURL := ap.tfClient.Client.BaseURL()
	assert.Equal(t, 1, countEnvVar(container1.Env, "TFC_AGENT_AUTO_UPDATE"))
	assert.Equal(t, "disabled", mustFindEnvVar(t, container1.Env, "TFC_AGENT_AUTO_UPDATE").Value)
	assert.Equal(t, 1, countEnvVar(container1.Env, "TFC_AGENT_TOKEN"))
	assert.Equal(
		t,
		agentPoolOutputObjectName(ap.instance.Name),
		mustFindEnvVar(t, container1.Env, "TFC_AGENT_TOKEN").ValueFrom.SecretKeyRef.Name,
	)
	assert.Equal(t, "token-a", mustFindEnvVar(t, container1.Env, "TFC_AGENT_TOKEN").ValueFrom.SecretKeyRef.Key)
	assert.Equal(t, 1, countEnvVar(container1.Env, "TFC_AGENT_NAME"))
	assert.Equal(t, "custom-agent-name", mustFindEnvVar(t, container1.Env, "TFC_AGENT_NAME").Value)
	assert.Equal(t, 1, countEnvVar(container1.Env, "TFC_ADDRESS"))
	assert.Equal(t, customURL.String(), mustFindEnvVar(t, container1.Env, "TFC_ADDRESS").Value)
}

func mustFindEnvVar(t *testing.T, envs []corev1.EnvVar, name string) corev1.EnvVar {
	t.Helper()

	for _, env := range envs {
		if env.Name == name {
			return env
		}
	}

	t.Fatalf("env var %q not found", name)
	return corev1.EnvVar{}
}

func countEnvVar(envs []corev1.EnvVar, name string) int {
	count := 0
	for _, env := range envs {
		if env.Name == name {
			count++
		}
	}

	return count
}

func TestUpdateDeployment_NoChangeSkipsUpdate(t *testing.T) {
	t.Parallel()

	ap := newTestAgentPoolInstance(t, "test-pool", "default")
	scheme := newTestScheme()

	// Build the desired deployment to use as the "existing" one
	existing := agentPoolDeployment(ap)
	existing.ResourceVersion = "12345"
	// Set owner reference as updateDeployment does
	existing.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         "app.terraform.io/v1alpha2",
			Kind:               "AgentPool",
			Name:               ap.instance.Name,
			UID:                ap.instance.UID,
			Controller:         pointer.PointerOf(true),
			BlockOwnerDeletion: pointer.PointerOf(true),
		},
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		Build()

	r := &AgentPoolReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(10),
	}

	err := r.updateDeployment(context.Background(), ap, existing)
	require.NoError(t, err)

	// Verify the deployment was NOT updated by checking ResourceVersion is unchanged
	var result appsv1.Deployment
	err = c.Get(context.Background(), types.NamespacedName{Name: existing.Name, Namespace: existing.Namespace}, &result)
	require.NoError(t, err)
	assert.Equal(t, "12345", result.ResourceVersion)
}

func TestUpdateDeployment_SpecChangeTriggersUpdate(t *testing.T) {
	t.Parallel()

	ap := newTestAgentPoolInstance(t, "test-pool", "default")
	scheme := newTestScheme()

	// Build the "existing" deployment with different replicas
	existing := agentPoolDeployment(ap)
	existing.ResourceVersion = "12345"
	existing.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         "app.terraform.io/v1alpha2",
			Kind:               "AgentPool",
			Name:               ap.instance.Name,
			UID:                ap.instance.UID,
			Controller:         pointer.PointerOf(true),
			BlockOwnerDeletion: pointer.PointerOf(true),
		},
	}
	// Change replicas on the existing deployment so it differs from desired
	existing.Spec.Replicas = pointer.PointerOf(int32(1))

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		Build()

	r := &AgentPoolReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(10),
	}

	err := r.updateDeployment(context.Background(), ap, existing)
	require.NoError(t, err)

	// Verify the deployment was updated — replicas should now be 3
	var result appsv1.Deployment
	err = c.Get(context.Background(), types.NamespacedName{Name: existing.Name, Namespace: existing.Namespace}, &result)
	require.NoError(t, err)
	assert.Equal(t, pointer.PointerOf(int32(3)), result.Spec.Replicas)
}

func TestUpdateDeployment_AnnotationChangeTriggersUpdate(t *testing.T) {
	t.Parallel()

	ap := newTestAgentPoolInstance(t, "test-pool", "default")
	scheme := newTestScheme()

	// Build the "existing" deployment with different annotations
	existing := agentPoolDeployment(ap)
	existing.ResourceVersion = "12345"
	existing.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         "app.terraform.io/v1alpha2",
			Kind:               "AgentPool",
			Name:               ap.instance.Name,
			UID:                ap.instance.UID,
			Controller:         pointer.PointerOf(true),
			BlockOwnerDeletion: pointer.PointerOf(true),
		},
	}
	// Change annotations on the existing deployment
	existing.Annotations["extra-annotation"] = "old-value"

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		Build()

	r := &AgentPoolReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(10),
	}

	err := r.updateDeployment(context.Background(), ap, existing)
	require.NoError(t, err)

	// Verify annotations were updated (extra-annotation should be gone)
	var result appsv1.Deployment
	err = c.Get(context.Background(), types.NamespacedName{Name: existing.Name, Namespace: existing.Namespace}, &result)
	require.NoError(t, err)
	_, hasExtra := result.Annotations["extra-annotation"]
	assert.False(t, hasExtra, "extra annotation should have been removed by update")
}

func TestUpdateDeployment_PreservesResourceVersion(t *testing.T) {
	t.Parallel()

	ap := newTestAgentPoolInstance(t, "test-pool", "default")
	scheme := newTestScheme()

	existing := agentPoolDeployment(ap)
	existing.ResourceVersion = "12345"
	existing.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion:         "app.terraform.io/v1alpha2",
			Kind:               "AgentPool",
			Name:               ap.instance.Name,
			UID:                ap.instance.UID,
			Controller:         pointer.PointerOf(true),
			BlockOwnerDeletion: pointer.PointerOf(true),
		},
	}
	// Force a change so the update path is taken
	existing.Spec.Replicas = pointer.PointerOf(int32(1))

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		Build()

	r := &AgentPoolReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: record.NewFakeRecorder(10),
	}

	err := r.updateDeployment(context.Background(), ap, existing)
	require.NoError(t, err)

	// The update should succeed because existing has ResourceVersion
	var result appsv1.Deployment
	err = c.Get(context.Background(), types.NamespacedName{Name: existing.Name, Namespace: existing.Namespace}, &result)
	require.NoError(t, err)
	assert.NotEmpty(t, result.ResourceVersion, "ResourceVersion should be preserved after update")
}

func TestAgentPoolDeployment_Deterministic(t *testing.T) {
	t.Parallel()

	ap := newTestAgentPoolInstance(t, "test-pool", "default")

	d1 := agentPoolDeployment(ap)
	d2 := agentPoolDeployment(ap)

	// Two calls with the same input should produce identical specs
	assert.True(t, cmp.Equal(d1.Spec, d2.Spec), "agentPoolDeployment should produce deterministic output")
	assert.True(t, cmp.Equal(d1.Annotations, d2.Annotations), "agentPoolDeployment annotations should be deterministic")
}

func TestAgentPoolDeployment_DefaultValues(t *testing.T) {
	t.Parallel()

	ap := newTestAgentPoolInstance(t, "test-pool", "default")
	ap.instance.Spec.AgentDeployment.Replicas = nil // use default

	d := agentPoolDeployment(ap)

	assert.Equal(t, pointer.PointerOf(int32(1)), d.Spec.Replicas, "should default to 1 replica")
	assert.Equal(t, &agentTerminationGracePeriod, d.Spec.Template.Spec.TerminationGracePeriodSeconds, "should set 15min termination grace period")
	assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, d.Spec.Strategy.Type)
	assert.Equal(t, pointer.PointerOf(intstr.FromInt(0)), d.Spec.Strategy.RollingUpdate.MaxSurge)
}
