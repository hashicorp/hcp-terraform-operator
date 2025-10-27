// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
	"maps"
	"slices"

	tfc "github.com/hashicorp/go-tfe"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
)

func (ap *agentPoolInstance) getTokens(ctx context.Context) (map[string]string, error) {
	agentTokens, err := ap.tfClient.Client.AgentTokens.List(ctx, ap.instance.Status.AgentPoolID)
	if err != nil {
		return nil, err
	}

	tokens := make(map[string]string)
	for _, token := range agentTokens.Items {
		tokens[token.ID] = token.Description
	}

	return tokens, nil
}

func (r *AgentPoolReconciler) createToken(ctx context.Context, ap *agentPoolInstance, token string) (*tfc.AgentToken, error) {
	// nn := getAgentPoolNamespacedName(&ap.instance)
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("creating a new agent token %q", token))
	t, err := ap.tfClient.Client.AgentTokens.Create(ctx, ap.instance.Status.AgentPoolID, tfc.AgentTokenCreateOptions{
		Description: &token,
	})
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to create a new token %q", token))
		return nil, err
	}
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully created a new agent token %q %q", token, t.ID))

	ap.instance.Status.AgentTokens = append(ap.instance.Status.AgentTokens, &appv1alpha2.AgentAPIToken{
		Name:       t.Description,
		ID:         t.ID,
		CreatedAt:  pointer.PointerOf(t.CreatedAt.Unix()),
		LastUsedAt: pointer.PointerOf(t.LastUsedAt.Unix()),
	})

	return t, nil
}

func (ap *agentPoolInstance) deleteTokenStatus(id string) {
	ap.instance.Status.AgentTokens = slices.DeleteFunc(ap.instance.Status.AgentTokens, func(vs *appv1alpha2.AgentAPIToken) bool {
		return vs.ID == id
	})
}

func agentPoolOutputObjectName(name string) string {
	return fmt.Sprintf("%s-agent-pool", name)
}

func getAgentPoolNamespacedName(instance *appv1alpha2.AgentPool) types.NamespacedName {
	return types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      agentPoolOutputObjectName(instance.Name),
	}
}

func (r *AgentPoolReconciler) createOrGetSecret(ctx context.Context, ap *agentPoolInstance) (*corev1.Secret, error) {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentPoolOutputObjectName(ap.instance.Name),
			Namespace: ap.instance.Namespace,
			Labels: map[string]string{
				"agentPoolID": ap.instance.Status.AgentPoolID,
			},
		},
	}
	if err := controllerutil.SetControllerReference(&ap.instance, s, r.Scheme); err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", "failed to set controller reference")
		return nil, err
	}
	if err := r.Client.Get(ctx, getAgentPoolNamespacedName(&ap.instance), s); err != nil {
		if errors.IsNotFound(err) {
			ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("creating a new Kubernetes Secret %q", s.Name))
			if err = r.Client.Create(ctx, s); err != nil {
				ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to create a new Kubernetes Secret %q", s.Name))
				return nil, err
			}
			ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully created a new Kubernetes Secret %q", s.Name))
			return s, nil
		}
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to get Kubernetes Secret %q", s.Name))
		return nil, err
	}

	return s, nil
}

func (r *AgentPoolReconciler) reconcileAgentTokens(ctx context.Context, ap *agentPoolInstance) error {
	ap.log.Info("Reconcile Agent Tokens", "msg", "new reconciliation event")

	s, err := r.createOrGetSecret(ctx, ap)
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to create a new Kubernetes Secret %s", agentPoolOutputObjectName(ap.instance.Name)))
		return err
	}
	if s.Data == nil {
		s.Data = make(map[string][]byte)
	}
	data := s.DeepCopy().Data

	agentTokens, err := ap.getTokens(ctx)
	if err != nil {
		return err
	}

	statusTokens := make(map[string]string, len(ap.instance.Status.AgentTokens))
	for _, t := range ap.instance.Status.AgentTokens {
		statusTokens[t.Name] = t.ID
	}

	for _, token := range ap.instance.Spec.AgentTokens {
		if id, ok := statusTokens[token.Name]; ok {
			delete(statusTokens, token.Name)
			if _, ok := agentTokens[id]; ok {
				delete(agentTokens, id)
				continue
			}
			delete(s.Data, token.Name)
			ap.deleteTokenStatus(id)
		}
		t, err := r.createToken(ctx, ap, token.Name)
		if err != nil {
			return err
		}
		s.Data[t.Description] = []byte(t.Token)
	}

	// Clean up.
	for name, id := range statusTokens {
		delete(s.Data, name)
		ap.deleteTokenStatus(id)
	}

	for id, name := range agentTokens {
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("removing agent token name=%q id=%q", name, id))
		err := ap.tfClient.Client.AgentTokens.Delete(ctx, id)
		if err != nil && err != tfc.ErrResourceNotFound {
			ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to remove agent token name=%q id=%q", name, id))
			return err
		}
		delete(s.Data, name)
		ap.deleteTokenStatus(id)
	}

	// Use defer to ensure the Secret is always updated, even if token creation or deletion fails.
	// This reduces the number of (Kubernetes) API calls and preserves the intermediate token state,
	// minimizing unnecessary updates during retries.
	defer func() {
		// Handle unexpected nil Secret, e.g. failed to retrieve it (should not happen here).
		if s == nil {
			return
		}
		// Do not update if there are no changes.
		if maps.EqualFunc(s.Data, data, func(vs, vd []byte) bool {
			return string(vs) == string(vd)
		}) {
			return
		}
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("updating Kubernetes Secret %q", s.Name))
		if err := r.Client.Update(ctx, s); err != nil {
			ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to update Kubernetes Secret %q", s.Name))
		}
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully updated Kubernetes Secret %q", s.Name))
	}()

	return nil
}
