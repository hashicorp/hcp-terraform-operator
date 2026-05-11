// Copyright IBM Corp. 2022, 2026
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tfc "github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func TestDecorateDeployment_MixedContainerEnvSets(t *testing.T) {
	t.Parallel()

	tfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/ping" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
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
