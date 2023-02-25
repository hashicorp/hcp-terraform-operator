// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AgentPoolReconciler) reconcileAgentDeployment(ctx context.Context, ap *agentPoolInstance) error {
	ap.log.Info("Reconcile Agent Deployment", "msg", "new reconciliation event")

	err := r.createDeployment(ctx, ap)
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Deployment", "msg", fmt.Sprintf("failed to create a new Kubernetes Deployment %s", agentPoolOutputObjectName(ap.instance.Name)))
		return err
	}
	return nil
}

func (r *AgentPoolReconciler) createDeployment(ctx context.Context, ap *agentPoolInstance) error {
	d := agentPoolDeployment(&ap.instance)
	err := controllerutil.SetControllerReference(&ap.instance, d, r.Scheme)
	if err != nil {
		return err
	}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: d.GetNamespace(), Name: d.GetName()}, d)
	if err != nil {
		if errors.IsAlreadyExists(err) {
			ap.log.Info("Reconcile Agent Deployment", "msg", fmt.Sprintf("Kubernetes Deployment %q exists", d.Name))
			return nil
		}
		if errors.IsNotFound(err) {
			ap.log.Info("Reconcile Agent Deployment", "msg", fmt.Sprintf("creating a new Kubernetes Deployment %q", d.Name))
			err = r.Client.Create(ctx, d)
			if err != nil {
				return err
			}
			ap.instance.Status.AgentDeploymentName = d.GetName()
			ap.log.Info("Reconcile Agent Deployment", "msg", fmt.Sprintf("successfully created a new Kubernetes Deployment %q", d.Name))
			return nil
		}
		ap.log.Error(err, "Reconcile Agent Deployment", "msg", fmt.Sprintf("failed to get Kubernetes Deployment %q", d.Name))
		return err
	}

	ap.log.Info("Reconcile Agent Deployment", "msg", fmt.Sprintf("Kubernetes Deployment %q exists", d.Name))
	return nil
}

func agentPoolDeployment(ap *v1alpha2.AgentPool) *appsv1.Deployment {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentPoolDeploymentName(ap),
			Namespace: ap.Namespace,
			Annotations: map[string]string{
				"app.terraform.io/agent-pool":    ap.Name,
				"app.terraform.io/agent-pool-id": ap.Status.AgentPoolID,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: agentPoolPodLabels(ap),
			},
			Replicas: ap.Spec.AgentDeployment.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: agentPoolPodLabels(ap),
				},
				Spec: ap.Spec.AgentDeployment.Spec,
			},
		},
	}
	// Inject a TFC_AGENT_TOKEN env var to each container in the Deployment.
	for ci := range d.Spec.Template.Spec.Containers {
		d.Spec.Template.Spec.Containers[ci].Env = append(d.Spec.Template.Spec.Containers[ci].Env,
			corev1.EnvVar{
				Name: "TFC_AGENT_TOKEN",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: agentPoolOutputObjectName(ap.Name)},
						Key:                  ap.Status.AgentTokens[0].Name,
					},
				},
			})
	}
	return d
}

func agentPoolDeploymentName(ap *v1alpha2.AgentPool) string {
	return fmt.Sprintf("agents-of-%s", ap.Name)
}

func agentPoolPodLabels(ap *v1alpha2.AgentPool) map[string]string {
	return map[string]string{
		"app.terraform.io/agent-pool": ap.Name,
	}
}
