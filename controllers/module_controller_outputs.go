package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func moduleOutputObjectName(name string) string {
	return fmt.Sprintf("%s-module-outputs", name)
}

func getModuleNamespacedName(instance *appv1alpha2.Module) types.NamespacedName {
	return types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      moduleOutputObjectName(instance.Name),
	}
}

// configMapAvailable validates whether a Kubernetes ConfigMap is available for creation or update by the operator
func (r *ModuleReconciler) configMapAvailable(ctx context.Context, instance *appv1alpha2.Module) bool {
	o := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, getModuleNamespacedName(instance), o)
	if err != nil {
		return errors.IsNotFound(err)
	}

	return containsOwnerReference(o.GetOwnerReferences(), instance.UID)
}

// secretAvailable validates whether a Kubernetes Secret is available for creation or update by the operator
func (r *ModuleReconciler) secretAvailable(ctx context.Context, instance *appv1alpha2.Module) bool {
	o := &corev1.Secret{}
	err := r.Client.Get(ctx, getModuleNamespacedName(instance), o)
	if err != nil {
		return errors.IsNotFound(err)
	}

	return containsOwnerReference(o.GetOwnerReferences(), instance.UID)
}

func (r *ModuleReconciler) setOutputs(ctx context.Context, instance *appv1alpha2.Module, workspace *tfc.Workspace) error {
	if workspace.CurrentStateVersion == nil {
		return nil
	}

	oName := moduleOutputObjectName(instance.Name)

	if !r.configMapAvailable(ctx, instance) {
		return fmt.Errorf("configMap %s is in use by different object thus it cannot be used to store outputs", oName)
	}

	if !r.secretAvailable(ctx, instance) {
		return fmt.Errorf("secret %s is in use by different object thus it cannot be used to store outputs", oName)
	}

	nonSensitiveOutput := make(map[string]string)
	sensitiveOutput := make(map[string][]byte)

	outputs, err := r.tfClient.Client.StateVersions.ListOutputs(ctx, workspace.CurrentStateVersion.ID, &tfc.StateVersionOutputsListOptions{})
	if err != nil {
		return err
	}
	for _, o := range outputs.Items {
		bytes, err := json.Marshal(o.Value)
		if err != nil {
			r.log.Error(err, "Reconcile Module Outputs", "mgs", fmt.Sprintf("failed to marshal JSON for %q", o.Name))
			r.Recorder.Event(instance, corev1.EventTypeWarning, "ReconcileOutputs", "failed to marshal JSON")
			continue
		}
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
		"workspaceID": instance.Status.WorkspaceID,
		"ModuleName":  instance.ObjectMeta.Name,
	}

	// update ConfigMap output
	cm := &corev1.ConfigMap{ObjectMeta: om}
	err = controllerutil.SetControllerReference(instance, cm, r.Scheme)
	if err != nil {
		return err
	}

	ur, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = labels
		cm.Data = nonSensitiveOutput
		return nil
	})
	if err != nil {
		r.log.Error(err, "Reconcile Module Outputs", "mgs", fmt.Sprintf("failed to create or update ConfigMap %s", oName))
		return err
	}
	r.log.Info("Reconcile Module Outputs", "mgs", fmt.Sprintf("configMap create or update result: %s", ur))

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
		r.log.Error(err, "Reconcile Module Outputs", "mgs", fmt.Sprintf("failed to create or update Secret %s", oName))
		return err
	}
	r.log.Info("Reconcile Module Outputs", "mgs", fmt.Sprintf("secret create or update result: %s", ur))

	return nil
}

func needToUpdateOutput(instance *appv1alpha2.Module) bool {
	status := instance.Status

	if status.Run == nil {
		return false
	}
	if status.Output == nil {
		return true
	}
	if status.Run.Status == string(tfc.RunApplied) && status.Run.ID != status.Output.RunID {
		return true
	}

	return false
}

func (r *ModuleReconciler) reconcileOutputs(ctx context.Context, instance *appv1alpha2.Module, workspace *tfc.Workspace) (string, error) {
	r.log.Info("Reconcile Module Outputs", "mgs", "new reconciliation event")

	if workspace.CurrentRun != nil {
		if needToUpdateOutput(instance) {
			r.log.Info("Reconcile Module Outputs", "mgs", "creating or updating outputs")
			return workspace.CurrentRun.ID, r.setOutputs(ctx, instance, workspace)
		}
	}
	return "", nil
}
