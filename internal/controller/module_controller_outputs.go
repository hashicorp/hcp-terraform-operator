// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/stateversions"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
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
		return kerrors.IsNotFound(err)
	}

	return containsOwnerReference(o.GetOwnerReferences(), instance.UID)
}

// secretAvailable validates whether a Kubernetes Secret is available for creation or update by the operator
func (r *ModuleReconciler) secretAvailable(ctx context.Context, instance *appv1alpha2.Module) bool {
	o := &corev1.Secret{}
	err := r.Client.Get(ctx, getModuleNamespacedName(instance), o)
	if err != nil {
		return kerrors.IsNotFound(err)
	}

	return containsOwnerReference(o.GetOwnerReferences(), instance.UID)
}

func (r *ModuleReconciler) setOutputs(ctx context.Context, m *moduleInstance) error {
	wsResp, err := m.tfClient.Client.API.Workspaces().ByWorkspace_id(m.instance.Status.WorkspaceID).Get(ctx, nil)
	if err != nil {
		return err
	}
	workspace := wsResp.GetData()
	sv := workspace.GetRelationships().GetCurrentStateVersion()
	if sv == nil || sv.GetData() == nil || pointer.ValueOf(sv.GetData().GetId()) == "" {
		return fmt.Errorf("current workspace state version is not available")
	}

	oName := moduleOutputObjectName(m.instance.Name)

	if !r.configMapAvailable(ctx, &m.instance) {
		return fmt.Errorf("configMap %s is in use by different object thus it cannot be used to store outputs", oName)
	}

	if !r.secretAvailable(ctx, &m.instance) {
		return fmt.Errorf("secret %s is in use by different object thus it cannot be used to store outputs", oName)
	}

	pageNumber := int32(InitPageNumber)
	var outputs []models.StateVersionOutputsable
	for {
		resp, err := m.tfClient.Client.API.StateVersions().
			ByState_version_id(pointer.ValueOf(sv.GetData().GetId())).
			Outputs().
			Get(ctx, &abstractions.RequestConfiguration[stateversions.ItemOutputsRequestBuilderGetQueryParameters]{
				QueryParameters: &stateversions.ItemOutputsRequestBuilderGetQueryParameters{
					Pagesize:   new(int32(MaxPageSize)),
					Pagenumber: &pageNumber,
				},
			})
		if err != nil {
			return err
		}
		outputs = append(outputs, resp.GetData()...)
		next := nextPage(resp.GetMeta())
		if next == 0 {
			break
		}
		pageNumber = next
	}

	nonSensitiveOutput := make(map[string]string)
	sensitiveOutput := make(map[string][]byte)
	for _, o := range outputs {
		name := pointer.ValueOf(o.GetAttributes().GetName())
		out, err := formatOutput(o)
		if err != nil {
			m.log.Error(err, "Reconcile Module Outputs", "mgs", fmt.Sprintf("failed to marshal JSON for %q", name))
			r.Recorder.Event(&m.instance, corev1.EventTypeWarning, "ReconcileOutputs", "failed to marshal JSON")
			continue
		}
		if pointer.ValueOf(o.GetAttributes().GetSensitive()) {
			sensitiveOutput[name] = []byte(out)
		} else {
			nonSensitiveOutput[name] = out
		}
	}

	om := metav1.ObjectMeta{
		Name:      oName,
		Namespace: m.instance.Namespace,
	}
	labels := map[string]string{
		"workspaceID": m.instance.Status.WorkspaceID,
	}

	// update ConfigMap output
	cm := &corev1.ConfigMap{ObjectMeta: om}
	err = controllerutil.SetControllerReference(&m.instance, cm, r.Scheme)
	if err != nil {
		return err
	}

	ur, err := controllerutil.CreateOrUpdate(ctx, r.Client, cm, func() error {
		cm.Labels = labels
		cm.Data = nonSensitiveOutput
		return nil
	})
	if err != nil {
		m.log.Error(err, "Reconcile Module Outputs", "mgs", fmt.Sprintf("failed to create or update ConfigMap %s", oName))
		return err
	}
	m.log.Info("Reconcile Module Outputs", "mgs", fmt.Sprintf("configMap create or update result: %s", ur))

	// update Secrets output
	secret := &corev1.Secret{ObjectMeta: om}
	err = controllerutil.SetControllerReference(&m.instance, secret, r.Scheme)
	if err != nil {
		return err
	}

	ur, err = controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Labels = labels
		secret.Data = sensitiveOutput
		return nil
	})
	if err != nil {
		m.log.Error(err, "Reconcile Module Outputs", "mgs", fmt.Sprintf("failed to create or update Secret %s", oName))
		return err
	}
	m.log.Info("Reconcile Module Outputs", "mgs", fmt.Sprintf("secret create or update result: %s", ur))

	return nil
}

func needToUpdateOutput(instance *appv1alpha2.Module) bool {
	status := instance.Status

	if status.Run == nil || !status.Run.RunApplied() {
		return false
	}

	return status.Output == nil || status.Output.RunID != status.Run.ID
}

func (r *ModuleReconciler) reconcileOutputs(ctx context.Context, m *moduleInstance, workspace models.Workspacesable) error {
	if cr := workspace.GetRelationships().GetCurrentRun(); cr != nil && cr.GetData() != nil && pointer.ValueOf(cr.GetData().GetId()) != "" {
		if needToUpdateOutput(&m.instance) {
			m.log.Info("Reconcile Module Outputs", "mgs", "creating or updating outputs")
			err := r.setOutputs(ctx, m)
			if err != nil {
				return err
			}
			m.instance.Status.Output = &appv1alpha2.OutputStatus{
				RunID: pointer.ValueOf(cr.GetData().GetId()),
			}
			return nil
		}
		m.log.Info("Reconcile Module Outputs", "mgs", "no need to update outputs")
	}
	return nil
}
