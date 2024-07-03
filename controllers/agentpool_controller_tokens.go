// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-cloud-operator/internal/pointer"
	"github.com/hashicorp/terraform-cloud-operator/internal/slice"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

func getAgentTokens(ctx context.Context, ap *agentPoolInstance) (map[string]struct{}, error) {
	agentTokens, err := ap.tfClient.Client.AgentTokens.List(ctx, ap.instance.Status.AgentPoolID)
	if err != nil {
		return nil, err
	}

	tokens := make(map[string]struct{}, len(agentTokens.Items))
	for _, at := range agentTokens.Items {
		tokens[at.ID] = struct{}{}
	}

	return tokens, nil
}

func (r *AgentPoolReconciler) createAgentToken(ctx context.Context, ap *agentPoolInstance, token string, secret *corev1.Secret) error {
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("creating a new agent token %q", token))
	at, err := ap.tfClient.Client.AgentTokens.Create(ctx, ap.instance.Status.AgentPoolID, tfc.AgentTokenCreateOptions{
		Description: &token,
	})
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to create a new token %q", token))
		return err
	}
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully created a new agent token %q %q", token, at.ID))

	ap.updateTokenStatus(&appv1alpha2.AgentToken{
		Name: at.Description,
		ID:   at.ID,
		// CreatedAt:  pointer.PointerOf(at.CreatedAt.Unix()),
		CreatedAt: pointer.PointerOf(int64(time.Now().Unix())),
		// LastUsedAt: pointer.PointerOf(at.LastUsedAt.Unix()),
		LastUsedAt: pointer.PointerOf(int64(0)),
	})

	labels := make(map[string]string)
	if secret.Labels != nil {
		labels = secret.DeepCopy().Labels
	}
	labels["agentPoolID"] = ap.instance.Status.AgentPoolID

	data := make(map[string][]byte)
	if secret.Data != nil {
		data = secret.DeepCopy().Data
	}
	data[at.Description] = []byte(at.Token)
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Labels = labels
		secret.Data = data
		return nil
	})
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to update Kubernets Secret %q with token %q", secret.Name, token))
		return err
	}
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully updated Kubernets Secret %q with token %q", secret.Name, token))

	return nil
}

func (ap *agentPoolInstance) removeTokenFromStatus(token string) {
	// TODO:
	// - we can re-write this function with slices.DeleteFunc.
	for i, st := range ap.instance.Status.AgentTokens {
		if st.ID == token {
			ap.instance.Status.AgentTokens = slice.RemoveFromSlice(ap.instance.Status.AgentTokens, i)
			return
		}
	}
}

func (ap *agentPoolInstance) updateTokenStatus(token *appv1alpha2.AgentToken) {
	// TODO:
	// - we can re-write this function with slices.IndexFunc.
	for i, st := range ap.instance.Status.AgentTokens {
		if st.ID == token.ID {
			ap.instance.Status.AgentTokens[i] = token
			return
		}
	}

	ap.instance.Status.AgentTokens = append(ap.instance.Status.AgentTokens, token)
}

func nameByTokenID(ap *agentPoolInstance, id string) string {
	// TODO:
	// - we can re-write this function with slices.IndexFunc.
	for _, st := range ap.instance.Status.AgentTokens {
		if st.ID == id {
			return st.Name
		}
	}

	return ""
}

func (r *AgentPoolReconciler) removeAgentToken(ctx context.Context, ap *agentPoolInstance, token string, secret *corev1.Secret) error {
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("removing agent token %q", token))
	err := ap.tfClient.Client.AgentTokens.Delete(ctx, token)
	if err != nil && err != tfc.ErrResourceNotFound {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to remove agent token %q", token))
		return err
	}
	n := nameByTokenID(ap, token)
	ap.removeTokenFromStatus(token)
	// Update Secret
	labels := make(map[string]string)
	if secret.Labels != nil {
		labels = secret.DeepCopy().Labels
	}
	labels["agentPoolID"] = ap.instance.Status.AgentPoolID

	data := make(map[string][]byte)
	if secret.Data != nil {
		data = secret.DeepCopy().Data
	}
	delete(data, n)
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Labels = labels
		secret.Data = data
		return nil
	})
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to remove token %q from Kubernets Secret %q", token, secret.Name))
		return err
	}
	ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully removed token %q from Kubernets Secret %q", token, secret.Name))

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

func (r *AgentPoolReconciler) getOrCreateSecret(ctx context.Context, ap *agentPoolInstance) (*corev1.Secret, error) {
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
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to set controller reference: %v", err))
		return nil, err
	}

	if err := r.Client.Get(ctx, getAgentPoolNamespacedName(&ap.instance), s); err != nil {
		if errors.IsNotFound(err) {
			ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("creating a new Kubernetes Secret %q", s.Name))
			if err := r.Client.Create(ctx, s); err != nil {
				ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to create a Kubernetes Secret %q", s.Name))
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

	secret, err := r.getOrCreateSecret(ctx, ap)
	if err != nil {
		ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to create a new Kubernetes Secret %s", agentPoolOutputObjectName(ap.instance.Name)))
		return err
	}

	// ROTATION::START
	if annotation, ok := ap.instance.Annotations[agentPoolAnnotationTokenRefreshDuration]; ok {
		refresh, err := time.ParseDuration(annotation)
		if err != nil {
			ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to parse annotation %v: %v", agentPoolAnnotationTokenRefreshDuration, annotation))
		} else {
			ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully parsed annotation %v: %v", agentPoolAnnotationTokenRefreshDuration, refresh))
			for _, token := range ap.instance.Status.AgentTokens {
				tokenDuration := time.Now().Unix() - *token.CreatedAt
				ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("token %v(%v) duration: %v", token.Name, token.ID, time.Duration(tokenDuration)*time.Second))
				if tokenDuration >= int64(refresh.Seconds()) && *token.LastUsedAt == 0 {
					ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("refreshing token %v(%v)", token.Name, token.ID))
					if err := r.createAgentToken(ctx, ap, token.Name, secret); err != nil {
						ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to refresh token %v(%v)", token.Name, token.ID))
						return err
					}
					token.LastUsedAt = pointer.PointerOf(time.Now().Unix())
				}
			}
		}
	}
	// ROTATION::END

	// Get HCP Terraform agent tokens.
	agentTokens, err := getAgentTokens(ctx, ap)
	if err != nil {
		return err
	}
	// Get instance spec agent tokens.
	specTokens := make(map[string]struct{}, len(ap.instance.Spec.AgentTokens))
	for _, t := range ap.instance.Spec.AgentTokens {
		specTokens[t.Name] = struct{}{}
	}
	// Get instance status agent tokens.
	activeTokens := make(map[string]string)
	retainedTokens := make(map[string]int64)
	for _, t := range ap.instance.Status.AgentTokens {
		if *t.LastUsedAt == 0 {
			activeTokens[t.Name] = t.ID
		} else {
			retainedTokens[t.ID] = *t.LastUsedAt
		}
	}

	for tokenName := range specTokens {
		if tokenID, ok := activeTokens[tokenName]; ok {
			if _, ok := agentTokens[tokenID]; ok {
				delete(agentTokens, tokenID)
				delete(activeTokens, tokenName)
				continue
			}
		}
		if err := r.createAgentToken(ctx, ap, tokenName, secret); err != nil {
			return err
		}
	}

	// Delete all tokens remained in HCP Terraform.
	for tokenID := range agentTokens {
		if _, ok := retainedTokens[tokenID]; !ok {
			if err := r.removeAgentToken(ctx, ap, tokenID, secret); err != nil {
				return err
			}
		}
	}

	// Clean up staus.
	for _, tokenID := range activeTokens {
		ap.removeTokenFromStatus(tokenID)
	}

	// Clean up retained tokens.
	retention := 0 * time.Second
	if annotation, ok := ap.instance.Annotations[agentPoolAnnotationTokenRetentionDuration]; ok {
		retention, err = time.ParseDuration(annotation)
		if err != nil {
			ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to parse annotation %v", agentPoolAnnotationTokenRetentionDuration))
		}
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully parsed annotation %v: %v", agentPoolAnnotationTokenRetentionDuration, retention))
	}
	timeNow := time.Now().Unix()
	for tokenID, lastUsedAt := range retainedTokens {
		if timeNow >= lastUsedAt+int64(retention.Seconds()) {
			ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("removing retained token %v", tokenID))
			// Remove retained token from HCP Terraform.
			err := ap.tfClient.Client.AgentTokens.Delete(ctx, tokenID)
			if err != nil && err != tfc.ErrResourceNotFound {
				ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to remove retained token %v", tokenID))
				return err
			}
			ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully removed retained token %v", tokenID))
			// Remove retained token from status.
			ap.removeTokenFromStatus(tokenID)
		}
	}

	return nil
}
