// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	corev1 "k8s.io/api/core/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *ProjectReconciler) deleteProject(ctx context.Context, p *projectInstance) error {
	p.log.Info("Reconcile Project", "msg", fmt.Sprintf("deletion policy is %s", p.instance.Spec.DeletionPolicy))

	if p.instance.Status.ID == "" {
		p.log.Info("Reconcile Project", "msg", fmt.Sprintf("status.ID is empty, remove finalizer %s", projectFinalizer))
		return r.removeFinalizer(ctx, p)
	}

	switch p.instance.Spec.DeletionPolicy {
	case appv1alpha2.ProjectDeletionPolicyRetain:
		p.log.Info("Reconcile Project", "msg", fmt.Sprintf("remove finalizer %s", projectFinalizer))
		return r.removeFinalizer(ctx, p)
	case appv1alpha2.ProjectDeletionPolicySoft:
		err := p.tfClient.Client.Projects.Delete(ctx, p.instance.Status.ID)
		if err != nil {
			if err == tfc.ErrResourceNotFound {
				p.log.Info("Reconcile Project", "msg", "Project was not found, remove finalizer")
				return r.removeFinalizer(ctx, p)
			}
			p.log.Error(err, "Reconcile Project", "msg", fmt.Sprintf("failed to delete project ID %s, retry later", p.instance.Status.ID))
			r.Recorder.Eventf(&p.instance, corev1.EventTypeWarning, "Reconcile Project", "Failed to destroy project ID %s, retry later", p.instance.Status.ID)
			return err
		}

		p.log.Info("Reconcile Project", "msg", fmt.Sprintf("Project ID %s has been deleted, remove finalizer", p.instance.Status.ID))
		return r.removeFinalizer(ctx, p)

	}

	return nil
}
