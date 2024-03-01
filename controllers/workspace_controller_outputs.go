// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func outputObjectName(name string) string {
	return fmt.Sprintf("%s-outputs", name)
}

func containsOwnerReference(ownerReferences []metav1.OwnerReference, UID types.UID) bool {
	for _, or := range ownerReferences {
		if or.UID == UID {
			return true
		}
	}

	return false
}

// configMapAvailable validates whether a Kubernetes ConfigMap is available for creation or update by the operator
func (r *WorkspaceReconciler) configMapAvailable(ctx context.Context, instance *appv1alpha2.Workspace) bool {
	o := &corev1.ConfigMap{}
	namespacedName := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      outputObjectName(instance.Name),
	}
	err := r.Client.Get(ctx, namespacedName, o)
	if err != nil {
		return errors.IsNotFound(err)
	}

	return containsOwnerReference(o.GetOwnerReferences(), instance.UID)
}

// secretAvailable validates whether a Kubernetes Secret is available for creation or update by the operator
func (r *WorkspaceReconciler) secretAvailable(ctx context.Context, instance *appv1alpha2.Workspace) bool {
	o := &corev1.Secret{}
	namespacedName := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      outputObjectName(instance.Name),
	}
	err := r.Client.Get(ctx, namespacedName, o)
	if err != nil {
		return errors.IsNotFound(err)
	}

	return containsOwnerReference(o.GetOwnerReferences(), instance.UID)
}

// func (r *WorkspaceReconciler) setOutputs(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
func (r *WorkspaceReconciler) setOutputs(ctx context.Context, w *workspaceInstance) error {
	workspace, err := w.tfClient.Client.Workspaces.ReadByID(ctx, w.instance.Status.WorkspaceID)
	if err != nil {
		return err
	}
	if workspace.CurrentStateVersion == nil {
		return fmt.Errorf("current workspace state version is not available")
	}

	oName := outputObjectName(w.instance.Name)

	if !r.configMapAvailable(ctx, &w.instance) {
		return fmt.Errorf("configMap %s is in use by different object thus it cannot be used to store outputs", oName)
	}

	if !r.secretAvailable(ctx, &w.instance) {
		return fmt.Errorf("secret %s is in use by different object thus it cannot be used to store outputs", oName)
	}

	outputs, err := w.tfClient.Client.StateVersions.ListOutputs(ctx, workspace.CurrentStateVersion.ID, &tfc.StateVersionOutputsListOptions{})
	if err != nil {
		return err
	}

	nonSensitiveOutput := make(map[string]string)
	sensitiveOutput := make(map[string][]byte)
	for _, o := range outputs.Items {
		out, err := formatOutput(o)
		if err != nil {
			w.log.Error(err, "Reconcile Module Outputs", "mgs", fmt.Sprintf("failed to marshal JSON for %q", o.Name))
			r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileOutputs", "failed to marshal JSON")
			continue
		}
		if o.Sensitive {
			sensitiveOutput[o.Name] = []byte(out)
		} else {
			nonSensitiveOutput[o.Name] = out
		}
	}

	om := metav1.ObjectMeta{
		Name:      oName,
		Namespace: w.instance.Namespace,
	}
	labels := map[string]string{
		"workspaceID":   w.instance.Status.WorkspaceID,
		"workspaceName": w.instance.Spec.Name,
	}

	// update ConfigMap output
	cm := &corev1.ConfigMap{ObjectMeta: om}
	err = controllerutil.SetControllerReference(&w.instance, cm, r.Scheme)
	if err != nil {
		return err
	}

	ur, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = labels
		cm.Data = nonSensitiveOutput
		return nil
	})
	if err != nil {
		w.log.Error(err, "Reconcile Outputs", "mgs", fmt.Sprintf("failed to create or update ConfigMap %s", oName))
		return err
	}
	w.log.Info("Reconcile Outputs", "mgs", fmt.Sprintf("configMap create or update result: %s", ur))

	// update Secrets output
	secret := &corev1.Secret{ObjectMeta: om}
	err = controllerutil.SetControllerReference(&w.instance, secret, r.Scheme)
	if err != nil {
		return err
	}

	ur, err = controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Labels = labels
		secret.Data = sensitiveOutput
		return nil
	})
	if err != nil {
		w.log.Error(err, "Reconcile Outputs", "mgs", fmt.Sprintf("failed to create or update Secret %s", oName))
		return err
	}
	w.log.Info("Reconcile Outputs", "mgs", fmt.Sprintf("secret create or update result: %s", ur))

	return nil
}

func (r *WorkspaceReconciler) runApplied(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) (bool, error) {
	run, err := w.tfClient.Client.Runs.Read(ctx, workspace.CurrentRun.ID)
	if err != nil {
		w.log.Error(err, "Reconcile Outputs", "mgs", fmt.Sprintf("failed to read current run ID %s", workspace.CurrentRun.ID))
		return false, err
	}

	if run.Status == tfc.RunApplied {
		return true, nil
	}

	return false, nil
}

func (r *WorkspaceReconciler) reconcileOutputs(ctx context.Context, w *workspaceInstance, workspace *tfc.Workspace) error {
	w.log.Info("Reconcile Outputs", "mgs", "new reconciliation event")

	if workspace.CurrentRun != nil {
		runApplied, err := r.runApplied(ctx, w, workspace)
		if err != nil {
			return err
		}
		if runApplied {
			w.log.Info("Reconcile Outputs", "mgs", "run successfully applied")
			if w.instance.Status.Run.OutputRunID != workspace.CurrentRun.ID {
				w.log.Info("Reconcile Outputs", "mgs", "creating or updating outputs")
				err = r.setOutputs(ctx, w)
				if err != nil {
					return err
				}
				w.instance.Status.Run.OutputRunID = workspace.CurrentRun.ID
				return nil
			}
			w.log.Info("Reconcile Outputs", "mgs", "no need to update outputs")
		}
	}
	return nil
}
