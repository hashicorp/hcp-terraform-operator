// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *AgentPoolReconciler) deleteAgentPool(ctx context.Context, ap *agentPoolInstance) error {
	ap.log.Info("Reconcile Agent Pool", "msg", fmt.Sprintf("deletion policy is %s", ap.instance.Spec.DeletionPolicy))

	if ap.instance.Status.AgentPoolID == "" {
		ap.log.Info("Reconcile Agent Pool", "msg", fmt.Sprintf("status.agentPoolID is empty, remove finalizer %s", agentPoolFinalizer))
		return r.removeFinalizer(ctx, ap)
	}

	switch ap.instance.Spec.DeletionPolicy {
	case appv1alpha2.AgentPoolDeletionPolicyRetain:
		ap.log.Info("Reconcile Agent Pool", "msg", fmt.Sprintf("remove finalizer %s", agentPoolFinalizer))
		return r.removeFinalizer(ctx, ap)
	case appv1alpha2.AgentPoolDeletionPolicy(appv1alpha2.DeletionPolicyDestroy):
		// Attempt to delete the agent pool first. If successful, no other actions are required.
		// Otherwise, scale down the agents to 0 and delete all tokens.
		err := ap.tfClient.Client.AgentPools.Delete(ctx, ap.instance.Status.AgentPoolID)
		if err != nil {
			// If agent pool wasn't found, it means it was deleted from the HCP Terraform bypass the operator.
			// In this case, remove the finalizer and let Kubernetes remove the object permanently
			if err == tfc.ErrResourceNotFound {
				ap.log.Info("Reconcile Agent Pool", "msg", fmt.Sprintf("agent pool ID %s not found, remove finalizer", agentPoolFinalizer))
				return r.removeFinalizer(ctx, ap)
			}
			ap.log.Error(err, "Reconcile Agent Pool", "msg", fmt.Sprintf("failed to delete Agent Pool ID %s, retry later", agentPoolFinalizer))
			r.Recorder.Eventf(&ap.instance, corev1.EventTypeWarning, "ReconcileAgentPool", "Failed to delete Agent Pool ID %s, retry later", ap.instance.Status.AgentPoolID)
			// Do not return the error here; proceed further to cale down the agents to 0 and delete all tokens.
		} else {
			ap.log.Info("Reconcile Agent Pool", "msg", fmt.Sprintf("agent pool ID %s has been deleted, remove finalizer", ap.instance.Status.AgentPoolID))
			return r.removeFinalizer(ctx, ap)
		}
		// Downscale agents
		if ap.instance.Status.AgentDeploymentAutoscalingStatus != nil && ap.instance.Status.AgentDeploymentAutoscalingStatus.DesiredReplicas != nil {
			if *ap.instance.Status.AgentDeploymentAutoscalingStatus.DesiredReplicas > 0 {
				ap.log.Info("Reconcile Agent Pool", "msg", fmt.Sprintf("scale agents from %d to 0", *ap.instance.Status.AgentDeploymentAutoscalingStatus.DesiredReplicas))
				var n int32 = 0
				if err := r.scaleAgentDeployment(ctx, ap, &n); err != nil {
					ap.log.Error(err, "Reconcile Agent Pool", "msg", "failed to scale agents")
					return err
				}
				ap.instance.Status.AgentDeploymentAutoscalingStatus = &appv1alpha2.AgentDeploymentAutoscalingStatus{
					DesiredReplicas: &n,
					LastScalingEvent: &metav1.Time{
						Time: time.Now(),
					},
				}
				ap.log.Info("Reconcile Agent Pool", "msg", "successfully scaled agents to 0")
			}
		}
		// Remove tokens
		if len(ap.instance.Status.AgentTokens) > 0 {
			ap.log.Info("Reconcile Agent Pool", "msg", "remove tokens")
			for _, t := range ap.instance.Status.AgentTokens {
				err := ap.tfClient.Client.AgentTokens.Delete(ctx, t.ID)
				if err != nil && err != tfc.ErrResourceNotFound {
					ap.log.Error(err, "Reconcile Agent Pool", "msg", fmt.Sprintf("failed to remove token %s", t.ID))
					return err
				}
				ap.deleteTokenStatus(t.ID)
			}
			ap.log.Info("Reconcile Agent Pool", "msg", "successfully deleted tokens")
		}
	}

	return nil
}
