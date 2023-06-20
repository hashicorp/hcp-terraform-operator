// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

func getWorkspacePendingRuns(ctx context.Context, ap *agentPoolInstance, workspaceID string) (int, error) {
	runs, err := ap.tfClient.Client.Runs.List(ctx, workspaceID, &tfc.RunListOptions{
		Status: string(tfc.RunPending),
	})
	if err != nil {
		return 0, err
	}
	return len(runs.Items), nil
}

func getTargetWorkspaceID(ctx context.Context, ap *agentPoolInstance, targetWorkspace appv1alpha2.TargetWorkspace) (string, error) {
	if targetWorkspace.ID != "" {
		return targetWorkspace.ID, nil
	}
	list, err := ap.tfClient.Client.Workspaces.List(ctx, ap.instance.Spec.Organization, &tfc.WorkspaceListOptions{
		Search: targetWorkspace.Name,
	})
	if err != nil {
		return "", err
	}
	for _, w := range list.Items {
		if w.Name == targetWorkspace.Name {
			return w.ID, nil
		}
	}
	return "", fmt.Errorf("no such workspace found %q", targetWorkspace.Name)
}

func getPendingRuns(ctx context.Context, ap *agentPoolInstance) (int, error) {
	workspaces := ap.instance.Spec.AgentDeploymentAutoscaling.TargetWorkspaces
	count := 0
	for _, w := range workspaces {
		id, err := getTargetWorkspaceID(ctx, ap, w)
		if err != nil {
			return 0, err
		}
		runs, err := getWorkspacePendingRuns(ctx, ap, id)
		if err != nil {
			return 0, err
		}
		count += runs
	}
	return count, nil
}

func getAgentDeploymentNamespacedName(ap *agentPoolInstance) types.NamespacedName {
	return types.NamespacedName{
		Namespace: ap.instance.Namespace,
		Name:      agentPoolDeploymentName(&ap.instance),
	}
}

func (r *AgentPoolReconciler) getAgentDeploymentReplicas(ctx context.Context, ap *agentPoolInstance) (*int32, error) {
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, getAgentDeploymentNamespacedName(ap), &deployment)
	if err != nil {
		return nil, err
	}
	return deployment.Spec.Replicas, nil
}

func (r *AgentPoolReconciler) scaleAgentDeployment(ctx context.Context, ap *agentPoolInstance, target *int32) error {
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, getAgentDeploymentNamespacedName(ap), &deployment)
	if err != nil {
		return err
	}
	deployment.Spec.Replicas = target
	return r.Client.Update(ctx, &deployment)
}

const defaultCooldownPeriodSeconds = 60

func (r *AgentPoolReconciler) reconcileAgentAutoscaling(ctx context.Context, ap *agentPoolInstance) error {
	if ap.instance.Spec.AgentDeploymentAutoscaling == nil {
		return nil
	}

	ap.log.Info("Reconcile Agent Autoscaling", "msg", "new reconciliation event")

	cooldownPeriodSeconds := ap.instance.Spec.AgentDeploymentAutoscaling.CooldownPeriodSeconds
	if cooldownPeriodSeconds == nil {
		cooldownPeriodSeconds = pointer.Int32(defaultCooldownPeriodSeconds)
	}

	status := ap.instance.Status.AgentDeploymentAutoscalingStatus
	if status != nil {
		lastScalingEvent := status.LastScalingEvent
		if lastScalingEvent != nil {
			lastScalingEventSeconds := int(time.Since(lastScalingEvent.Time).Seconds())
			if lastScalingEventSeconds <= int(*cooldownPeriodSeconds) {
				ap.log.Info("Reconcile Agent Autoscaling", "msg", "autoscaler is within the cooldown period, skipping")
				return nil
			}
		}
	}

	pendingRuns, err := getPendingRuns(ctx, ap)
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to get pending runs")
		r.Recorder.Event(&ap.instance, corev1.EventTypeWarning, "Autoscaling failed", err.Error())
		return err
	}

	currentReplicas, err := r.getAgentDeploymentReplicas(ctx, ap)
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to get current replicas")
		r.Recorder.Event(&ap.instance, corev1.EventTypeWarning, "Autoscaling failed", err.Error())
		return err
	}

	ap.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("Pending runs: %v", pendingRuns))
	ap.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("Current replicas: %v", *currentReplicas))

	newReplicas := currentReplicas
	if pendingRuns == 0 {
		newReplicas = ap.instance.Spec.AgentDeploymentAutoscaling.MinReplicas
	} else if (int(*currentReplicas) + pendingRuns) > int(*ap.instance.Spec.AgentDeploymentAutoscaling.MaxReplicas) {
		newReplicas = ap.instance.Spec.AgentDeploymentAutoscaling.MaxReplicas
	} else if pendingRuns > int(*currentReplicas) {
		newReplicas = pointer.Int32(int32(int(*currentReplicas) + pendingRuns))
	}

	if *newReplicas != *currentReplicas {
		ap.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("Scaling agent deployment from %v to %v", *currentReplicas, *newReplicas))
		r.Recorder.Event(&ap.instance, corev1.EventTypeNormal, "autoscaling", "scaling agent deployment")
		err := r.scaleAgentDeployment(ctx, ap, newReplicas)
		if err != nil {
			ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to scale agent deployment")
			r.Recorder.Event(&ap.instance, corev1.EventTypeWarning, "Autoscaling failed", err.Error())
			return err
		}
	}
	return nil
}
