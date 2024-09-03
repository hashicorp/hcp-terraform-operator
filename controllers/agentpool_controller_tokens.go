// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
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

func (r *AgentPoolReconciler) createAgentPoolToken(ctx context.Context, ap *agentPoolInstance, token string) error {
	nn := getAgentPoolNamespacedName(&ap.instance)
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("creating a new agent token %q", token))
	at, err := ap.tfClient.Client.AgentTokens.Create(ctx, ap.instance.Status.AgentPoolID, tfc.AgentTokenCreateOptions{
		Description: &token,
	})
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to create a new token %q", token))
		return err
	}
	ap.instance.Status.AgentTokens = append(ap.instance.Status.AgentTokens, &appv1alpha2.AgentToken{
		Name:       at.Description,
		ID:         at.ID,
		CreatedAt:  pointer.PointerOf(at.CreatedAt.Unix()),
		LastUsedAt: pointer.PointerOf(at.LastUsedAt.Unix()),
	})
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully created a new agent token %q %q", token, at.ID))
	// UPDATE SECRET
	s := &corev1.Secret{}
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("update Kubernets Secret %q with token %q", s.Name, token))
	if err := r.Client.Get(ctx, nn, s); err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to get Kubernets Secret %q", s.Name))
		return err
	}
	d := make(map[string][]byte)
	if s.Data != nil {
		d = s.DeepCopy().Data
	}
	d[at.Description] = []byte(at.Token)
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, s, func() error {
		s.Data = d
		s.Labels = map[string]string{
			"agentPoolID": ap.instance.Status.AgentPoolID,
		}
		return nil
	})
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to update Kubernets Secret %q with token %q", s.Name, token))
		return nil
	}
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully updated Kubernets Secret %q with token %q", s.Name, token))

	return nil
}

func removeTokenFromStatus(ap *agentPoolInstance, t string) {
	var statusTokens []*appv1alpha2.AgentToken
	for _, token := range ap.instance.Status.AgentTokens {
		if token.ID != t {
			statusTokens = append(statusTokens, token)
		}
	}

	ap.instance.Status.AgentTokens = statusTokens
}

func nameByTokenID(ap *agentPoolInstance, id string) string {
	for _, token := range ap.instance.Status.AgentTokens {
		if token.ID == id {
			return token.Name
		}
	}

	return ""
}

func (r *AgentPoolReconciler) removeAgentPoolToken(ctx context.Context, ap *agentPoolInstance, token string) error {
	nn := getAgentPoolNamespacedName(&ap.instance)
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("removing agent token %q", token))
	err := ap.tfClient.Client.AgentTokens.Delete(ctx, token)
	if err != nil && err != tfc.ErrResourceNotFound {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to remove agent token %q", token))
		return err
	}
	n := nameByTokenID(ap, token)
	removeTokenFromStatus(ap, token)
	// UPDATE SECRET
	s := &corev1.Secret{}
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("remove token %q from Kubernets Secret %q", token, nn.Name))
	if err := r.Client.Get(ctx, nn, s); err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to get Kubernets Secret %q", nn.Name))
		return err
	}
	d := make(map[string][]byte)
	if s.Data != nil {
		d = s.DeepCopy().Data
	}
	delete(d, n)
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, s, func() error {
		s.Data = d
		s.Labels = map[string]string{
			"agentPoolID": ap.instance.Status.AgentPoolID,
		}
		return nil
	})
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to remove token %q from Kubernets Secret %q", token, s.Name))
		return nil
	}
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully removed token %q from Kubernets Secret %q", token, s.Name))

	return nil
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

func (r *AgentPoolReconciler) createSecret(ctx context.Context, ap *agentPoolInstance) error {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentPoolOutputObjectName(ap.instance.Name),
			Namespace: ap.instance.Namespace,
			Labels: map[string]string{
				"agentPoolID": ap.instance.Status.AgentPoolID,
			},
		},
	}
	err := controllerutil.SetControllerReference(&ap.instance, s, r.Scheme)
	if err != nil {
		return err
	}
	err = r.Client.Get(ctx, getAgentPoolNamespacedName(&ap.instance), s)
	if err != nil {
		if errors.IsNotFound(err) {
			ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("creating a new Kubernetes Secret %q", s.Name))
			err = r.Client.Create(ctx, s)
			if err != nil {
				return err
			}
			ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully created a new Kubernetes Secret %q", s.Name))
			return nil
		}
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to get Kubernetes Secret %q", s.Name))
		return err
	}

	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("Kubernetes Secret %q exists", s.Name))
	return nil
}

func (r *AgentPoolReconciler) reconcileAgentTokens(ctx context.Context, ap *agentPoolInstance) error {
	ap.log.Info("Reconcile Agent Tokens", "msg", "new reconciliation event")

	err := r.createSecret(ctx, ap)
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to create a new Kubernetes Secret %s", agentPoolOutputObjectName(ap.instance.Name)))
		return err
	}

	agentTokens, err := ap.getTokens(ctx)
	if err != nil {
		return err
	}

	statusTokens := make(map[string]string, len(ap.instance.Status.AgentTokens))
	for _, t := range ap.instance.Status.AgentTokens {
		statusTokens[t.Name] = t.ID
	}

	for _, token := range ap.instance.Spec.AgentTokens {
		if tokenID, ok := statusTokens[token.Name]; ok {
			if _, ok := agentTokens[tokenID]; ok {
				delete(agentTokens, tokenID)
			} else {
				removeTokenFromStatus(ap, tokenID)
				if err := r.createAgentPoolToken(ctx, ap, token.Name); err != nil {
					return err
				}
			}
		} else {
			if err := r.createAgentPoolToken(ctx, ap, token.Name); err != nil {
				return err
			}
		}
	}

	for tokenID := range agentTokens {
		r.removeAgentPoolToken(ctx, ap, tokenID)
	}

	return nil
}
