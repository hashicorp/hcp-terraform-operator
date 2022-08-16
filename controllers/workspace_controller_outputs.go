package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

func getNamespacedName(instance *appv1alpha2.Workspace) types.NamespacedName {
	return types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      outputObjectName(instance.Name),
	}
}

func (r *WorkspaceReconciler) canCreateOrUseConfigMap(ctx context.Context, instance *appv1alpha2.Workspace) bool {
	o := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, getNamespacedName(instance), o)
	if err != nil {
		return errors.IsNotFound(err)
	}

	return containsOwnerReference(o.GetOwnerReferences(), instance.UID)
}

func (r *WorkspaceReconciler) canCreateOrUseSecret(ctx context.Context, instance *appv1alpha2.Workspace) bool {
	o := &corev1.Secret{}
	err := r.Client.Get(ctx, getNamespacedName(instance), o)
	if err != nil {
		return errors.IsNotFound(err)
	}

	return containsOwnerReference(o.GetOwnerReferences(), instance.UID)
}

func trimDoubleQuotes(bytes []byte) []byte {
	s := string(bytes)
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	return []byte(s)
}

func (r *WorkspaceReconciler) updateOrCreateOutputs(ctx context.Context, instance *appv1alpha2.Workspace, workspace *tfc.Workspace) error {
	oName := outputObjectName(instance.Name)

	if !r.canCreateOrUseConfigMap(ctx, instance) {
		return fmt.Errorf("configMap %s is in use by different object thus it cannot be used to store outputs", oName)
	}

	if !r.canCreateOrUseSecret(ctx, instance) {
		return fmt.Errorf("secret %s is in use by different object thus it cannot be used to store outputs", oName)
	}

	nonSensitiveOutput := make(map[string]string)
	sensitiveOutput := make(map[string][]byte)

	outputs, _ := r.tfClient.Client.StateVersions.ListOutputs(ctx, workspace.CurrentStateVersion.ID, &tfc.StateVersionOutputsListOptions{})
	for _, o := range outputs.Items {
		bytes, _ := json.Marshal(o.Value)
		if o.Sensitive {
			sensitiveOutput[o.Name] = trimDoubleQuotes(bytes)
		} else {
			nonSensitiveOutput[o.Name] = string(trimDoubleQuotes(bytes))
		}
	}

	om := metav1.ObjectMeta{
		Name:      oName,
		Namespace: instance.Namespace,
	}
	labels := map[string]string{
		"workspaceID":   instance.Status.WorkspaceID,
		"workspaceName": instance.Spec.Name,
	}

	// update ConfigMap output
	cm := &corev1.ConfigMap{ObjectMeta: om}
	err := controllerutil.SetControllerReference(instance, cm, r.Scheme)
	if err != nil {
		return err
	}

	ur, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = labels
		cm.Data = nonSensitiveOutput
		return nil
	})
	if err != nil {
		r.log.Error(err, "Reconcile Outputs", "mgs", fmt.Sprintf("failed to create or update ConfigMap %s", oName))
		return err
	}
	r.log.Info("Reconcile Outputs", "mgs", fmt.Sprintf("configMap create or update result: %s", ur))

	// update Secrets output
	secret := &corev1.Secret{ObjectMeta: om}
	err = controllerutil.SetControllerReference(instance, secret, r.Scheme)
	if err != nil {
		return err
	}

	ur, err = controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Labels = labels
		secret.Data = sensitiveOutput
		return nil
	})
	if err != nil {
		r.log.Error(err, "Reconcile Outputs", "mgs", fmt.Sprintf("failed to create or update Secret %s", oName))
		return err
	}
	r.log.Info("Reconcile Outputs", "mgs", fmt.Sprintf("secret create or update result: %s", ur))

	return nil
}

func (r *WorkspaceReconciler) hasRunApplied(ctx context.Context, workspace *tfc.Workspace) (bool, error) {
	run, err := r.tfClient.Client.Runs.Read(ctx, workspace.CurrentRun.ID)
	if err != nil {
		r.log.Error(err, "Reconcile Outputs", "mgs", fmt.Sprintf("failed to read current run ID %s", workspace.CurrentRun.ID))
		return false, err
	}

	if run.Status == tfc.RunApplied {
		return true, nil
	}

	return false, nil
}

func (r *WorkspaceReconciler) reconcileOutputs(ctx context.Context, instance *appv1alpha2.Workspace, workspace *tfc.Workspace) error {
	r.log.Info("Reconcile Outputs", "mgs", "new reconciliation event")

	if workspace.CurrentRun != nil {
		runFinished, err := r.hasRunApplied(ctx, workspace)
		if err != nil {
			return err
		}
		if runFinished && instance.Status.OutputRunID != workspace.CurrentRun.ID {
			r.log.Info("Reconcile Outputs", "mgs", "creating or updating outputs")
			return r.updateOrCreateOutputs(ctx, instance, workspace)
		}
	}
	return nil
}
