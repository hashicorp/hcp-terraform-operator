// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

func (r *AgentTokenReconciler) deleteAgentToken(ctx context.Context, t *agentTokenInstance) error {
	t.log.Info("Reconcile Agent Token", "msg", fmt.Sprintf("deletion policy is %s", t.instance.Spec.DeletionPolicy))

	if t.instance.Status.AgentPool == nil && t.instance.Status.AgentPool.ID == "" {
		t.log.Info("Reconcile Agent Token", "msg", fmt.Sprintf("Agent Pool ID is not set, remove finalizer %s", agentPoolFinalizer))
		return r.removeFinalizer(ctx, t)
	}

	switch t.instance.Spec.DeletionPolicy {
	case appv1alpha2.AgentTokenDeletionPolicyRetain:
		nn := types.NamespacedName{
			Namespace: t.instance.Namespace,
			Name:      t.instance.Spec.Token.SecretKeyRef.Name,
		}
		s := &corev1.Secret{}
		if err := r.Client.Get(ctx, nn, s); err != nil {
			if apierrors.IsNotFound(err) {
				return r.removeFinalizer(ctx, t)
			}
			t.log.Error(err, "Reconcile Agent Token", "msg", fmt.Sprintf("failed to get secret=%q namespace=%q", nn.Name, nn.Namespace))
			return err
		}
		if err := controllerutil.RemoveControllerReference(&t.instance, s, r.Scheme); err != nil {
			t.log.Error(err, "Reconcile Agent Token", "msg", fmt.Sprintf("failed to remove controller reference from secret=%q namespace=%q", nn.Name, nn.Namespace))
			return err
		}
		return r.removeFinalizer(ctx, t)
	case appv1alpha2.AgentTokenDeletionPolicyDestroy:
		if len(t.instance.Status.AgentTokens) > 0 {
			t.log.Info("Reconcile Agent Token", "msg", "remove tokens")
			for _, token := range t.instance.Status.AgentTokens {
				err := t.tfClient.Client.AgentTokens.Delete(ctx, token.ID)
				if err != nil && err != tfc.ErrResourceNotFound {
					t.log.Error(err, "Reconcile Agent Pool", "msg", fmt.Sprintf("failed to remove token %s", token.ID))
					return err
				}
				err = r.removeToken(ctx, t, token.ID)
				if err != nil {
					return err
				}
			}
			t.log.Info("Reconcile Agent Pool", "msg", "successfully deleted tokens")
		}
	}

	return nil
}
