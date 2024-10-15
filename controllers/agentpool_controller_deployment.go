// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"
	"net/url"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	poolNameLabel             = "agentpool.app.terraform.io/pool-name"
	poolIDLabel               = "agentpool.app.terraform.io/pool-id"
	defaultAgentImage         = "hashicorp/tfc-agent"
	defaultAgentContainerName = "tfc-agent"
	deploymentAnnotations     = "agentpool.app.terraform.io/deployment-name"
)

func (r *AgentPoolReconciler) reconcileAgentDeployment(ctx context.Context, ap *agentPoolInstance) error {
	ap.log.Info("Reconcile Agent Deployment", "msg", "new reconciliation event")
	var d *appsv1.Deployment = &appsv1.Deployment{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: ap.instance.Namespace, Name: agentPoolDeploymentName(&ap.instance)}, d)
	if err == nil {
		if ap.instance.Spec.AgentDeployment == nil {
			// Delete the existing deployment
			return r.deleteDeployment(ctx, ap, d)
		}
		// Update existing deployment
		return r.updateDeployment(ctx, ap)
	}
	if errors.IsNotFound(err) {
		if ap.instance.Spec.AgentDeployment == nil { // Was a deployment configured?
			ap.log.Info("Reconcile Agent Deployment", "msg",
				fmt.Sprintf("skipping - no deployment configured in AgentPool %q", ap.instance.GetName()))
			return nil
		}
		// Create deployment
		cerr := r.createDeployment(ctx, ap)
		if cerr != nil {
			ap.log.Error(cerr, "Reconcile Agent Deployment", "msg", fmt.Sprintf("failed to create a new Kubernetes Deployment %s", agentPoolOutputObjectName(ap.instance.Name)))
			r.Recorder.Event(&ap.instance, corev1.EventTypeWarning, "Deployment creation failed", cerr.Error())
			return cerr
		}
		return nil
	}
	ap.log.Error(err, "Reconcile Agent Deployment", "msg", fmt.Sprintf("failed to get Kubernetes Deployment %q", d.Name))
	return err

}

func (r *AgentPoolReconciler) createDeployment(ctx context.Context, ap *agentPoolInstance) error {
	if len(ap.instance.Status.AgentTokens) < 1 {
		return fmt.Errorf("not enough tokens available")
	}
	d := agentPoolDeployment(ap)
	err := controllerutil.SetControllerReference(&ap.instance, d, r.Scheme)
	if err != nil {
		return err
	}
	// if autoscaler is enabled, set the replicas to the min
	if a := ap.instance.Spec.AgentDeploymentAutoscaling; a != nil {
		d.Spec.Replicas = a.MinReplicas
	}
	ap.log.Info("Reconcile Agent Deployment", "msg", fmt.Sprintf("creating a new Kubernetes Deployment %q", d.Name))
	err = r.Client.Create(ctx, d, &client.CreateOptions{FieldManager: "hcp-terraform-operator"})
	if err != nil {
		return err
	}
	ap.instance.Status.AgentDeploymentName = d.GetName()
	ap.log.Info("Reconcile Agent Deployment", "msg", fmt.Sprintf("successfully created a new Kubernetes Deployment %q", d.Name))
	return nil
}

func (r *AgentPoolReconciler) updateDeployment(ctx context.Context, ap *agentPoolInstance) error {
	ap.log.Info("Reconcile Agent Deployment", "mgs", "performing Deployment update")
	nd := agentPoolDeployment(ap)
	err := controllerutil.SetControllerReference(&ap.instance, nd, r.Scheme)
	if err != nil {
		return err
	}
	// if autoscaler is enabled, set the replicas to the desired replica count
	if a := ap.instance.Status.AgentDeploymentAutoscalingStatus; a != nil {
		nd.Spec.Replicas = a.DesiredReplicas
	}
	uerr := r.Client.Update(ctx, nd, &client.UpdateOptions{FieldManager: "hcp-terraform-operator"})
	if uerr != nil {
		ap.log.Error(uerr, "Reconcile Agent Deployment", "msg", "Failed to update agent deployment")
		r.Recorder.Event(&ap.instance, corev1.EventTypeWarning, "Deployment update failed", uerr.Error())
		return uerr
	}
	return nil
}

func (r *AgentPoolReconciler) deleteDeployment(ctx context.Context, ap *agentPoolInstance, d *appsv1.Deployment) error {
	ap.log.Info("Reconcile Agent Deployment", "msg",
		fmt.Sprintf("deleting agent deployment for AgentPool %q", ap.instance.GetName()))
	derr := r.Client.Delete(ctx, d)
	if derr != nil {
		if errors.IsNotFound(derr) {
			return nil
		}
		ap.log.Error(derr, "Reconcile Agent Deployment", "msg", fmt.Sprintf("failed to delete agent deployment '%s' for agent pool: %s", d.Name, agentPoolOutputObjectName(ap.instance.Name)))
		r.Recorder.Event(&ap.instance, corev1.EventTypeWarning, "Deployment deletion failed", derr.Error())
		return derr
	}
	ap.log.Info("Reconcile Agent Deployment", "msg", fmt.Sprintf("successfully deleted agent deployment %q", d.Name))
	r.Recorder.Event(&ap.instance, corev1.EventTypeNormal, "Deployment creation succeeded", "")
	return nil
}

var agentTerminationGracePeriod int64 = 900 // 15 minutes

func agentPoolDeployment(ap *agentPoolInstance) *appsv1.Deployment {
	var r *int32 = pointer.PointerOf(int32(1)) // default to one replica if not otherwise configured
	var agentPodLabels = agentPodLabels(&ap.instance)
	if ap.instance.Spec.AgentDeployment.Replicas != nil {
		r = ap.instance.Spec.AgentDeployment.Replicas
	}
	// don't set the replica count if autoscaling is enabled
	if ap.instance.Spec.AgentDeploymentAutoscaling != nil {
		r = nil
	}
	var s corev1.PodSpec = corev1.PodSpec{
		Containers: []corev1.Container{ // default tfc-agent container if none configured by user
			{
				Name:  defaultAgentContainerName,
				Image: defaultAgentImage,
			},
		},
	}
	if ap.instance.Spec.AgentDeployment.Spec != nil {
		s = *ap.instance.Spec.AgentDeployment.Spec
	}
	if s.TerminationGracePeriodSeconds == nil {
		// set a more sensible termination grace period to allow
		// agents still active to complete their run
		s.TerminationGracePeriodSeconds = &agentTerminationGracePeriod
	}
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentPoolDeploymentName(&ap.instance),
			Namespace: ap.instance.Namespace,
			Annotations: map[string]string{
				poolNameLabel: ap.instance.Name,
				poolIDLabel:   ap.instance.Status.AgentPoolID,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: agentPodLabels,
			},
			Replicas: r,
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					// this is important to ensure the number of pods does not temporarily
					// shoot over the max agents allowed when rolling the deployment.
					MaxSurge: pointer.PointerOf(intstr.FromInt(0)),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      agentPodLabels,
					Annotations: agentPodAnnotations(&ap.instance),
				},
				Spec: s,
			},
		},
	}
	decorateDeployment(ap, d)
	return d
}

func decorateDeployment(ap *agentPoolInstance, d *appsv1.Deployment) {
	envs := []corev1.EnvVar{
		{
			Name: "TFC_AGENT_TOKEN",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: agentPoolOutputObjectName(ap.instance.Name)},
					Key:                  ap.instance.Status.AgentTokens[0].Name,
				},
			},
		},
		{
			Name: "TFC_AGENT_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name:  "TFC_AGENT_AUTO_UPDATE",
			Value: "disabled",
		},
	}
	// Set TFE_ADDRESS on agent Pod if differnet than default TFC endpoint.
	bURL := ap.tfClient.Client.BaseURL()
	if defURL, perr := url.Parse(tfc.DefaultAddress); perr == nil && defURL.Host != bURL.Host {
		envs = append(envs, corev1.EnvVar{
			Name:  "TFC_ADDRESS",
			Value: bURL.String(),
		})
	}
	// Inject agent specific environment vars to each container in the Deployment.
	for ci := range d.Spec.Template.Spec.Containers {
		d.Spec.Template.Spec.Containers[ci].Env = append(d.Spec.Template.Spec.Containers[ci].Env, envs...)
	}
}

func agentPoolDeploymentName(ap *appv1alpha2.AgentPool) string {
	return fmt.Sprintf("agents-of-%s", ap.Name)
}

func agentPodLabels(ap *appv1alpha2.AgentPool) map[string]string {

	label := map[string]string{}

	//Attempting to merge
	if len(ap.Spec.AgentDeployment.Labels) > 0 {
		for key, value := range ap.Spec.AgentDeployment.Labels {
			label[key] = value
		}
	}

	//built in assignment
	label[poolNameLabel] = ap.Name

	return label
}

func agentPodAnnotations(ap *appv1alpha2.AgentPool) map[string]string {

	annotations := map[string]string{}

	if len(ap.Spec.AgentDeployment.Annotations) > 0 {
		for key, value := range ap.Spec.AgentDeployment.Annotations {
			annotations[key] = value
		}
	}

	annotations[deploymentAnnotations] = ap.Name

	return annotations
}
