package controllers

import (
	"context"
	"fmt"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func getAgentPoolTokens(ctx context.Context, ap *agentPoolInstance) (map[string]bool, error) {
	t := make(map[string]bool)

	agentTokens, err := ap.tfClient.Client.AgentTokens.List(ctx, ap.instance.Status.AgentPoolID)
	if err != nil {
		return nil, err
	}
	for _, at := range agentTokens.Items {
		t[at.ID] = true
	}

	return t, nil
}

func getTokensToRemove(ctx context.Context, ap *agentPoolInstance) ([]string, error) {
	var d []string
	// diff spec and status
	s := make(map[string]bool, len(ap.instance.Spec.AgentTokens))
	for _, t := range ap.instance.Spec.AgentTokens {
		s[t.Name] = true
	}

	for _, t := range ap.instance.Status.AgentTokens {
		if !s[t.Name] {
			d = append(d, t.ID)
		}
	}

	// diff status and agent pool
	st := make(map[string]bool, len(ap.instance.Status.AgentTokens))
	for _, t := range ap.instance.Status.AgentTokens {
		st[t.ID] = true
	}
	agentPoolTokens, err := getAgentPoolTokens(ctx, ap)
	if err != nil {
		return nil, err
	}
	for t := range st {
		if !agentPoolTokens[t] {
			d = append(d, t)
		}
	}

	return d, nil
}

func getTokensToCreate(ctx context.Context, ap *agentPoolInstance) (map[string]bool, error) {
	a := make(map[string]bool)

	// remove token from state if they don't exist in TFC
	at, err := getAgentPoolTokens(ctx, ap)
	if err != nil {
		return a, err
	}

	for _, t := range ap.instance.Status.AgentTokens {
		if !at[t.ID] {
			removeTokenFromStatus(ap, t.ID)
		}
	}

	// spec and status diff
	st := make(map[string]bool, len(ap.instance.Status.AgentTokens))
	for _, t := range ap.instance.Status.AgentTokens {
		st[t.Name] = true
	}

	for _, t := range ap.instance.Spec.AgentTokens {
		if !st[t.Name] {
			a[t.Name] = true
		}
	}

	return a, nil
}

func (r *AgentPoolReconciler) createAgentPoolTokens(ctx context.Context, ap *agentPoolInstance, tokens map[string]bool) error {
	for t := range tokens {
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("creating a new agent token %q", t))
		at, err := ap.tfClient.Client.AgentTokens.Create(ctx, ap.instance.Status.AgentPoolID, tfc.AgentTokenCreateOptions{
			Description: &t,
		})
		if err != nil {
			ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to create a new token %q", t))
			return err
		}
		ap.instance.Status.AgentTokens = append(ap.instance.Status.AgentTokens, &appv1alpha2.AgentToken{
			Name:       at.Description,
			ID:         at.ID,
			CreatedAt:  at.CreatedAt.Unix(),
			LastUsedAt: at.LastUsedAt.Unix(),
		})
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully created a new agent token %q %q", t, at.ID))
		// UPDATE SECRET
		s := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      agentPoolOutputObjectName(ap.instance.Name),
				Namespace: ap.instance.Namespace,
			},
		}
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("update Kubernets Secret %q with token %q", s.Name, t))
		err = r.Client.Get(ctx, getAgentPoolNamespacedName(&ap.instance), s)
		if err != nil {
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
			ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to update Kubernets Secret %q with token %q", s.Name, t))
			return nil
		}
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully updated Kubernets Secret %q with token %q", s.Name, t))
	}

	return nil
}

func removeTokenFromStatus(ap *agentPoolInstance, t string) {
	var s []*appv1alpha2.AgentToken
	for _, st := range ap.instance.Status.AgentTokens {
		if st.ID != t {
			s = append(s, st)
		}
	}

	ap.instance.Status.AgentTokens = s
}

func nameByTokenID(ap *agentPoolInstance, id string) string {
	for _, st := range ap.instance.Status.AgentTokens {
		if st.ID == id {
			return st.Name
		}
	}

	return ""
}

func (r *AgentPoolReconciler) removeAgentPoolTokens(ctx context.Context, ap *agentPoolInstance, tokens []string) error {
	for _, t := range tokens {
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("removing agent token %q", t))
		err := ap.tfClient.Client.AgentTokens.Delete(ctx, t)
		if err != nil && err != tfc.ErrResourceNotFound {
			ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to remove agent token %q", t))
			return err
		}
		n := nameByTokenID(ap, t)
		removeTokenFromStatus(ap, t)
		// UPDATE SECRET
		s := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      agentPoolOutputObjectName(ap.instance.Name),
				Namespace: ap.instance.Namespace,
			},
		}
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("remove token %q from Kubernets Secret %q", t, s.Name))
		err = r.Client.Get(ctx, getAgentPoolNamespacedName(&ap.instance), s)
		if err != nil {
			ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to get Kubernets Secret %q", s.Name))
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
			ap.log.Error(err, "Reconcile Agent Tokens", "msg", fmt.Sprintf("failed to remove token %q from Kubernets Secret %q", t, s.Name))
			return nil
		}
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("successfully removed token %q from Kubernets Secret %q", t, s.Name))
	}

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

	removeTokens, err := getTokensToRemove(ctx, ap)
	if err != nil {
		return err
	}
	if len(removeTokens) > 0 {
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("removing %d agent tokens from the agent pool", len(removeTokens)))
		err := r.removeAgentPoolTokens(ctx, ap, removeTokens)
		if err != nil {
			return err
		}
	}

	createTokens, err := getTokensToCreate(ctx, ap)
	if err != nil {
		return err
	}
	if len(createTokens) > 0 {
		ap.log.Info("Reconcile Agent Tokens", "msg", fmt.Sprintf("creating %d agent tokens in the agent pool", len(createTokens)))
		err := r.createAgentPoolTokens(ctx, ap, createTokens)
		if err != nil {
			return err
		}
	}

	return nil
}
