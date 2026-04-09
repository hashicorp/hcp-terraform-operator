// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
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

func newTestAgentPoolInstance(name, namespace string) *agentPoolInstance {
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
		log: logr.Discard(),
		tfClient: func() HCPTerraformClient {
			c, _ := tfc.NewClient(&tfc.Config{Token: "test-token"})
			return HCPTerraformClient{Client: c}
		}(),
	}
}

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = appv1alpha2.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	return s
}

func TestUpdateDeployment_NoChangeSkipsUpdate(t *testing.T) {
	t.Parallel()

	ap := newTestAgentPoolInstance("test-pool", "default")
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

	ap := newTestAgentPoolInstance("test-pool", "default")
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

	ap := newTestAgentPoolInstance("test-pool", "default")
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

	ap := newTestAgentPoolInstance("test-pool", "default")
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

	ap := newTestAgentPoolInstance("test-pool", "default")

	d1 := agentPoolDeployment(ap)
	d2 := agentPoolDeployment(ap)

	// Two calls with the same input should produce identical specs
	assert.True(t, cmp.Equal(d1.Spec, d2.Spec), "agentPoolDeployment should produce deterministic output")
	assert.True(t, cmp.Equal(d1.Annotations, d2.Annotations), "agentPoolDeployment annotations should be deterministic")
}

func TestAgentPoolDeployment_DefaultValues(t *testing.T) {
	t.Parallel()

	ap := newTestAgentPoolInstance("test-pool", "default")
	ap.instance.Spec.AgentDeployment.Replicas = nil // use default

	d := agentPoolDeployment(ap)

	assert.Equal(t, pointer.PointerOf(int32(1)), d.Spec.Replicas, "should default to 1 replica")
	assert.Equal(t, &agentTerminationGracePeriod, d.Spec.Template.Spec.TerminationGracePeriodSeconds, "should set 15min termination grace period")
	assert.Equal(t, appsv1.RollingUpdateDeploymentStrategyType, d.Spec.Strategy.Type)
	assert.Equal(t, pointer.PointerOf(intstr.FromInt(0)), d.Spec.Strategy.RollingUpdate.MaxSurge)
}
