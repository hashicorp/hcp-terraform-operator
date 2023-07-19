// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"
	"strings"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

func getWorkspaceQueueDepth(ctx context.Context, ap *agentPoolInstance, workspaceID string) (int, error) {
	statuses := []string{
		string(tfc.RunPending),
		string(tfc.RunPlanQueued),
		string(tfc.RunApplyQueued),
	}
	runs, err := ap.tfClient.Client.Runs.List(ctx, workspaceID, &tfc.RunListOptions{
		Status: strings.Join(statuses, ","),
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

func getQueueDepth(ctx context.Context, ap *agentPoolInstance) (int, error) {
	workspaces := ap.instance.Spec.AgentDeploymentAutoscaling.TargetWorkspaces
	depth := 0
	for _, w := range workspaces {
		id, err := getTargetWorkspaceID(ctx, ap, w)
		if err != nil {
			return 0, err
		}
		runs, err := getWorkspaceQueueDepth(ctx, ap, id)
		if err != nil {
			return 0, err
		}
		depth += runs
	}
	return depth, nil
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

func (r *AgentPoolReconciler) reconcileAgentAutoscaling(ctx context.Context, ap *agentPoolInstance) error {
	if ap.instance.Spec.AgentDeploymentAutoscaling == nil {
		return nil
	}

	ap.log.Info("Reconcile Agent Autoscaling", "msg", "new reconciliation event")

	status := ap.instance.Status.AgentDeploymentAutoscalingStatus
	if status != nil {
		lastScalingEvent := status.LastScalingEvent
		if lastScalingEvent != nil {
			lastScalingEventSeconds := int(time.Since(lastScalingEvent.Time).Seconds())
			cooldownPeriodSeconds := ap.instance.Spec.AgentDeploymentAutoscaling.CooldownPeriodSeconds
			if lastScalingEventSeconds <= int(*cooldownPeriodSeconds) {
				ap.log.Info("Reconcile Agent Autoscaling", "msg", "autoscaler is within the cooldown period, skipping")
				return nil
			}
		}
	}

	queueDepth, err := getQueueDepth(ctx, ap)
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to get pending runs")
		r.Recorder.Eventf(&ap.instance, corev1.EventTypeWarning, "AutoscaleAgentPoolDeployment", "Autoscaling failed: %v", err.Error())
		return err
	}

	currentReplicas, err := r.getAgentDeploymentReplicas(ctx, ap)
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to get current replicas")
		r.Recorder.Eventf(&ap.instance, corev1.EventTypeWarning, "AutoscaleAgentPoolDeployment", "Autoscaling failed: %v", err.Error())
		return err
	}

	desiredReplicas := currentReplicas
	if queueDepth == 0 {
		desiredReplicas = ap.instance.Spec.AgentDeploymentAutoscaling.MinReplicas
	} else if (int(*currentReplicas) + queueDepth) > int(*ap.instance.Spec.AgentDeploymentAutoscaling.MaxReplicas) {
		desiredReplicas = ap.instance.Spec.AgentDeploymentAutoscaling.MaxReplicas
	} else if queueDepth > int(*currentReplicas) {
		desiredReplicas = pointer.Int32(int32(int(*currentReplicas) + queueDepth))
	}

	if *desiredReplicas != *currentReplicas {
		scalingEvent := fmt.Sprintf("Scaling agent deployment from %v to %v replicas", *currentReplicas, *desiredReplicas)
		ap.log.Info("Reconcile Agent Autoscaling", "msg", scalingEvent)
		r.Recorder.Event(&ap.instance, corev1.EventTypeNormal, "AutoscaleAgentPoolDeployment", scalingEvent)
		err := r.scaleAgentDeployment(ctx, ap, desiredReplicas)
		if err != nil {
			ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to scale agent deployment")
			r.Recorder.Eventf(&ap.instance, corev1.EventTypeWarning, "AutoscaleAgentPoolDeployment", "Autoscaling failed: %v", err.Error())
			return err
		}
		ap.instance.Status.AgentDeploymentAutoscalingStatus = &appv1alpha2.AgentDeploymentAutoscalingStatus{
			DesiredReplicas: desiredReplicas,
			LastScalingEvent: &v1.Time{
				Time: time.Now(),
			},
		}
		r.updateStatus(ctx, ap, nil)
	}
	return nil
}
