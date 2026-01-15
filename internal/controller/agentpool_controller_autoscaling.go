// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

// userInteractionRunStatuses contains run statuses that require user interaction.
var userInteractionRunStatuses = map[tfc.RunStatus]struct{}{
	tfc.RunCostEstimated:            {},
	tfc.RunPlanned:                  {},
	tfc.RunPlannedAndSaved:          {},
	tfc.RunPolicyOverride:           {},
	tfc.RunPostPlanAwaitingDecision: {},
	tfc.RunPostPlanCompleted:        {},
	tfc.RunPending:                  {},
}

// matchWildcardName checks if a given string matches a specified wildcard pattern.
// The wildcard pattern can contain '*' at the beginning and/or end to match any sequence of characters.
// If the pattern contains '*' at both ends, the function checks if the substring exists within the string.
// If the pattern contains '*' only at the beginning, the function checks if the string ends with the substring.
// If the pattern contains '*' only at the end, the function checks if the string starts with the substring.
// If there are no '*' characters, the function checks for an exact match.
// For example:
// (1) '*-terraform-workspace' -- the wildcard indicator '*' is at the beginning of the wildcard name (prefix),
// therefore, we should search for a workspace name that ends with the suffix '-terraform-workspace'.
// (2) 'hcp-terraform-workspace-*' -- the wildcard indicator '*' is at the end of the wildcard name (suffix),
// therefore, we should search for a workspace name that starts with the prefix 'hcp-terraform-workspace-'.
// (3) '*-terraform-workspace-*' -- the wildcard indicator '*' is at the beginning and the end of the wildcard name (prefix and suffix),
// therefore, we should search for a workspace name containing the substring '-terraform-workspace-'.
func matchWildcardName(wildcard string, str string) bool {
	// Both 'prefix' and 'suffix' indicate whether a part of the name is in the prefix, suffix, or both.
	// If the wildcard indicator '*' is in the PREFIX part, then search for a substring that is in the SUFFIX.
	// If the wildcard indicator '*' is in the SUFFIX part, then search for a substring that is in the PREFIX.
	// If the wildcard indicator '*' is in both the prefix and the suffix, then search for a substring that is in between '*'.
	prefix := strings.HasSuffix(wildcard, "*")
	suffix := strings.HasPrefix(wildcard, "*")
	wn := strings.Trim(wildcard, "*")
	switch {
	case prefix && suffix:
		return strings.Contains(str, wn)
	case prefix:
		return strings.HasPrefix(str, wn)
	case suffix:
		return strings.HasSuffix(str, wn)
	default:
		return wn == str
	}
}

// pendingRuns returns the number pending runs for a given agent pool.
// This function is compatible with HCP Terraform and TFE version v202409-1 and later.
func pendingRuns(ctx context.Context, ap *agentPoolInstance) (int32, error) {
	applyRuns := map[string]struct{}{}
	awaitingUserInteractionRuns := map[string]int{} // Track runs awaiting user interaction by status for future metrics
	listOpts := &tfc.RunListForOrganizationOptions{
		AgentPoolNames: ap.instance.Spec.Name,
		StatusGroup:    "non_final",
		ListOptions: tfc.ListOptions{
			PageSize:   MaxPageSize,
			PageNumber: InitPageNumber,
		},
	}
	planOnlyRuns := 0
	for {
		runsList, err := ap.tfClient.Client.Runs.ListForOrganization(ctx, ap.instance.Spec.Organization, listOpts)
		if err != nil {
			return 0, err
		}

		for _, run := range runsList.Items {
			// Skip runs that require user interaction
			if _, ok := userInteractionRunStatuses[run.Status]; ok {
				// Save the user interactable run statuses for future metrics with count split by status
				awaitingUserInteractionRuns[string(run.Status)]++
				continue
			}
			// Count plan-only runs separately so agents can scale up and execute runs parallely
			if run.PlanOnly {
				planOnlyRuns++
				continue
			}
			applyRuns[run.Workspace.ID] = struct{}{}
		}
		if runsList.NextPage == 0 {
			break
		}
		listOpts.PageNumber = runsList.NextPage
	}

	// TODO:
	// Add metric(s) for runs awaiting user interaction
	totalPendingRuns := len(applyRuns) + planOnlyRuns
	ap.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("apply/plan-only runs: %d/%d", len(applyRuns), planOnlyRuns))
	return int32(totalPendingRuns), nil
}

// computeRequiredAgents is a legacy algorithm that is used to compute the number of agents needed.
// It is used when the TFE version is less than v202409-1.
func computeRequiredAgents(ctx context.Context, ap *agentPoolInstance) (int32, error) {
	required := 0
	// NOTE:
	// - Two maps are used here to simplify target workspace searching by ID, name, and wildcard.
	workspaceNames := map[string]struct{}{}
	workspaceIDs := map[string]struct{}{}

	listOpts := &tfc.WorkspaceListOptions{
		CurrentRunStatus: strings.Join([]string{
			string(tfc.RunPlanQueued),
			string(tfc.RunApplyQueued),
			string(tfc.RunApplying),
			string(tfc.RunPlanning),
		}, ","),
		ListOptions: tfc.ListOptions{
			PageSize:   MaxPageSize,
			PageNumber: InitPageNumber,
		},
	}
	for {
		workspaceList, err := ap.tfClient.Client.Workspaces.List(ctx, ap.instance.Spec.Organization, listOpts)
		if err != nil {
			return 0, err
		}
		for _, ws := range workspaceList.Items {
			if ws.AgentPool != nil && ws.AgentPool.ID == ap.instance.Status.AgentPoolID {
				workspaceNames[ws.Name] = struct{}{}
				workspaceIDs[ws.ID] = struct{}{}
			}
		}
		if workspaceList.NextPage == 0 {
			break
		}
		listOpts.PageNumber = workspaceList.NextPage
	}

	if ap.instance.Spec.AgentDeploymentAutoscaling.TargetWorkspaces == nil {
		return int32(len(workspaceNames)), nil
	}

	for _, t := range *ap.instance.Spec.AgentDeploymentAutoscaling.TargetWorkspaces {
		switch {
		case t.Name != "":
			if _, ok := workspaceNames[t.Name]; ok {
				required++
				delete(workspaceNames, t.Name)
			}
		case t.ID != "":
			if _, ok := workspaceIDs[t.ID]; ok {
				required++
			}
		case t.WildcardName != "":
			for w := range workspaceNames {
				if ok := matchWildcardName(t.WildcardName, w); ok {
					required++
					delete(workspaceNames, w)
				}
			}
		}
	}

	return int32(required), nil
}

func computeDesiredReplicas(requiredAgents, minReplicas, maxReplicas int32) int32 {
	if requiredAgents <= minReplicas {
		return minReplicas
	} else if requiredAgents >= maxReplicas {
		return maxReplicas
	}
	return requiredAgents
}

func getAgentDeploymentNamespacedName(ap *agentPoolInstance) types.NamespacedName {
	return types.NamespacedName{
		Namespace: ap.instance.Namespace,
		Name:      AgentPoolDeploymentName(&ap.instance),
	}
}

func (r *AgentPoolReconciler) getAgentDeploymentReplicas(ctx context.Context, ap *agentPoolInstance) (int32, error) {
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, getAgentDeploymentNamespacedName(ap), &deployment)
	if err != nil {
		return 0, err
	}
	return *deployment.Spec.Replicas, nil
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

// cooldownSecondsRemaining returns the remaining seconds in the Cool Down stage.
// A negative value indicates expired Cool Down.
func (a *agentPoolInstance) cooldownSecondsRemaining(currentReplicas, desiredReplicas int32) int {
	status := a.instance.Status.AgentDeploymentAutoscalingStatus
	if status == nil || status.LastScalingEvent == nil {
		return -1
	}

	cooldownPeriodSeconds := int(*a.instance.Spec.AgentDeploymentAutoscaling.CooldownPeriodSeconds)

	cooldownPeriod := a.instance.Spec.AgentDeploymentAutoscaling.CooldownPeriod
	if cooldownPeriod != nil {
		if v := cooldownPeriod.ScaleDownSeconds; v != nil {
			cooldownPeriodSeconds = int(*v)
			if currentReplicas > desiredReplicas {
				a.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("Agents scaling down, using configured scale down period: %v", cooldownPeriodSeconds))
			}
		}

		if v := cooldownPeriod.ScaleUpSeconds; v != nil {
			if desiredReplicas > currentReplicas {
				cooldownPeriodSeconds = int(*v)
				a.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("Agents scaling up, using configured scale up period: %v", cooldownPeriodSeconds))
			}
		}
	}

	lastScalingEventSeconds := int(time.Since(status.LastScalingEvent.Time).Seconds())
	return cooldownPeriodSeconds - lastScalingEventSeconds
}

func (r *AgentPoolReconciler) reconcileAgentAutoscaling(ctx context.Context, ap *agentPoolInstance) error {
	if ap.instance.Spec.AgentDeploymentAutoscaling == nil {
		return nil
	}

	ap.log.Info("Reconcile Agent Autoscaling", "msg", "new reconciliation event")

	requiredAgents, err := func() (int32, error) {
		if ap.tfClient.Client.IsCloud() {
			return pendingRuns(ctx, ap)
		}
		tfeVersion := ap.tfClient.Client.RemoteTFEVersion()
		runsEndpoint, err := useRunsEndpoint(tfeVersion)
		if err != nil {
			// If the TFE version parsing fails, do not return the error here and proceed further.
			// In this case, a legacy algorithm will be taken.
			ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to parse TFE version")
			r.Recorder.Eventf(&ap.instance, corev1.EventTypeWarning, "AutoscaleAgentPool", "Failed to parse TFE version: %v", err.Error())
		}
		// In TFE version v202409-1, a new Runs API endpoint was introduced.
		// It now allows retrieving a list of runs for the organization.
		if runsEndpoint {
			ap.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("Proceeding with the new algorithm based on the detected TFE version %s", tfeVersion))
			return pendingRuns(ctx, ap)
		}
		ap.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("Proceeding with the legacy algorithm based to the detected TFE version %s", tfeVersion))
		return computeRequiredAgents(ctx, ap)
	}()
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to get agents needed")
		r.Recorder.Eventf(&ap.instance, corev1.EventTypeWarning, "AutoscaleAgentPool", "Failed to get agents needed: %v", err.Error())
		return err
	}
	ap.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("%d agents are required", requiredAgents))

	currentReplicas, err := r.getAgentDeploymentReplicas(ctx, ap)
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to get current replicas")
		r.Recorder.Eventf(&ap.instance, corev1.EventTypeWarning, "AutoscaleAgentPool", "Failed to get current replicas: %v", err.Error())
		return err
	}
	ap.log.Info("Reconcile Agent Autoscaling", "msg", fmt.Sprintf("%d agent replicas are running", currentReplicas))

	minReplicas := *ap.instance.Spec.AgentDeploymentAutoscaling.MinReplicas
	maxReplicas := *ap.instance.Spec.AgentDeploymentAutoscaling.MaxReplicas
	desiredReplicas := computeDesiredReplicas(requiredAgents, minReplicas, maxReplicas)

	if desiredReplicas != currentReplicas {
		if ap.cooldownSecondsRemaining(currentReplicas, desiredReplicas) > 0 {
			ap.log.Info("Reconcile Agent Autoscaling", "msg", "autoscaler is within the cooldown period, skipping")
			return nil
		}

		scalingEvent := fmt.Sprintf("Scaling agent deployment from %v to %v replicas", currentReplicas, desiredReplicas)
		ap.log.Info("Reconcile Agent Autoscaling", "msg", strings.ToLower(scalingEvent))
		r.Recorder.Event(&ap.instance, corev1.EventTypeNormal, "AutoscaleAgentPool", scalingEvent)
		err := r.scaleAgentDeployment(ctx, ap, &desiredReplicas)
		if err != nil {
			ap.log.Error(err, "Reconcile Agent Autoscaling", "msg", "Failed to scale agent deployment")
			r.Recorder.Eventf(&ap.instance, corev1.EventTypeWarning, "AutoscaleAgentPool", "Failed to scale agent deployment: %v", err.Error())
			return err
		}
		ap.instance.Status.AgentDeploymentAutoscalingStatus = &appv1alpha2.AgentDeploymentAutoscalingStatus{
			DesiredReplicas: &desiredReplicas,
			LastScalingEvent: &metav1.Time{
				Time: time.Now(),
			},
		}
	}

	if ap.instance.Status.AgentDeploymentAutoscalingStatus == nil {
		ap.instance.Status.AgentDeploymentAutoscalingStatus = &appv1alpha2.AgentDeploymentAutoscalingStatus{
			DesiredReplicas: &desiredReplicas,
		}
	}

	return nil
}
