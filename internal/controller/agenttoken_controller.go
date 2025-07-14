// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/go-logr/logr"
	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/version"
)

// AgentTokenReconciler reconciles a AgentToken object
type AgentTokenReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

type agentTokenInstance struct {
	instance appv1alpha2.AgentToken

	log      logr.Logger
	tfClient HCPTerraformClient
}

//+kubebuilder:rbac:groups=apt.terraform.io,resources=agenttokens,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apt.terraform.io,resources=agenttokens/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apt.terraform.io,resources=agenttokens/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *AgentTokenReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	t := agentTokenInstance{}

	t.log = log.Log.WithValues("agenttoken", req.NamespacedName)
	t.log.Info("Agent Token Controller", "msg", "new reconciliation event")

	err := r.Client.Get(ctx, req.NamespacedName, &t.instance)
	if err != nil {
		// 'Not found' error occurs when an object is removed from the Kubernetes
		// No actions are required in this case
		if errors.IsNotFound(err) {
			t.log.Info("Agent Token Controller", "msg", "the instance was removed no further action is required")
			return doNotRequeue()
		}
		t.log.Error(err, "Agent Token Controller", "msg", "get instance object")
		return requeueAfter(requeueInterval)
	}

	t.log.Info("Spec Validation", "msg", "validating instance object spec")
	if err := t.instance.ValidateSpec(); err != nil {
		t.log.Error(err, "Spec Validation", "msg", "spec is invalid, exit from reconciliation")
		r.Recorder.Event(&t.instance, corev1.EventTypeWarning, "SpecValidation", err.Error())
		return doNotRequeue()
	}
	t.log.Info("Spec Validation", "msg", "spec is valid")

	if needToAddFinalizer(&t.instance, agentTokenFinalizer) {
		err := r.addFinalizer(ctx, &t.instance)
		if err != nil {
			t.log.Error(err, "Agent Token Controller", "msg", fmt.Sprintf("failed to add finalizer %s to the object", agentTokenFinalizer))
			r.Recorder.Eventf(&t.instance, corev1.EventTypeWarning, "AddFinalizer", "Failed to add finalizer %s to the object", agentTokenFinalizer)
			return requeueOnErr(err)
		}
		t.log.Info("Agent Token Controller", "msg", fmt.Sprintf("successfully added finalizer %s to the object", agentTokenFinalizer))
		r.Recorder.Eventf(&t.instance, corev1.EventTypeNormal, "AddFinalizer", "Successfully added finalizer %s to the object", agentTokenFinalizer)
	}

	err = r.getTerraformClient(ctx, &t)
	if err != nil {
		t.log.Error(err, "Agent Token Controller", "msg", "failed to get HCP Terraform client")
		r.Recorder.Event(&t.instance, corev1.EventTypeWarning, "TerraformClient", "Failed to get HCP Terraform Client")
		return requeueAfter(requeueInterval)
	}

	err = r.reconcileToken(ctx, &t)
	if err != nil {
		t.log.Error(err, "Agent Token Controller", "msg", "Reconcile Agent Token")
		r.Recorder.Event(&t.instance, corev1.EventTypeWarning, "ReconcileAgentToken", "Failed to Reconcile Agent Token")
		return requeueAfter(requeueInterval)
	}
	t.log.Info("Agent Token Controller", "msg", "successfully reconcilied agent token")
	r.Recorder.Event(&t.instance, corev1.EventTypeNormal, "ReconcileAgentToken", "Successfully reconcilied agent token")

	return requeueAfter(AgentTokenSyncPeriod)
}

func (r *AgentTokenReconciler) addFinalizer(ctx context.Context, instance *appv1alpha2.AgentToken) error {
	controllerutil.AddFinalizer(instance, agentTokenFinalizer)

	return r.Update(ctx, instance)
}

func (r *AgentTokenReconciler) getTerraformClient(ctx context.Context, t *agentTokenInstance) error {
	nn := types.NamespacedName{
		Namespace: t.instance.Namespace,
		Name:      t.instance.Spec.Token.SecretKeyRef.Name,
	}
	token, err := secretKeyRef(ctx, r.Client, nn, t.instance.Spec.Token.SecretKeyRef.Key)
	if err != nil {
		return err
	}

	httpClient := tfc.DefaultConfig().HTTPClient
	insecure := false

	if v, ok := os.LookupEnv("TFC_TLS_SKIP_VERIFY"); ok {
		insecure, err = strconv.ParseBool(v)
		if err != nil {
			return err
		}
	}

	if insecure {
		t.log.Info("Reconcile Agent Token", "msg", "client configured to skip TLS certificate verifications")
	}

	httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}

	config := &tfc.Config{
		Token:      token,
		HTTPClient: httpClient,
		Headers: http.Header{
			"User-Agent": []string{version.UserAgent},
		},
	}
	t.tfClient.Client, err = tfc.NewClient(config)

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentTokenReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha2.AgentToken{}).
		WithEventFilter(predicate.Or(genericPredicates())).
		Complete(r)
}

func (r *AgentTokenReconciler) removeFinalizer(ctx context.Context, t *agentTokenInstance) error {
	controllerutil.RemoveFinalizer(&t.instance, agentTokenFinalizer)

	err := r.Update(ctx, &t.instance)
	if err != nil {
		t.log.Error(err, "Reconcile Agent Token", "msg", fmt.Sprintf("failed to remove finalizer %s", agentTokenFinalizer))
		r.Recorder.Eventf(&t.instance, corev1.EventTypeWarning, "RemoveAgentToken", "Failed to remove finalizer %s", agentTokenFinalizer)
	}

	return err
}

func (r *AgentTokenReconciler) reconcileToken(ctx context.Context, t *agentTokenInstance) error {
	t.log.Info("Reconcile Agent Token", "msg", "reconciling agent token")

	// verify whether the Kubernetes object has been marked as deleted and if so delete the project
	if isDeletionCandidate(&t.instance, agentTokenFinalizer) {
		t.log.Info("Reconcile Agent Token", "msg", "object marked as deleted, need to delete project first")
		r.Recorder.Event(&t.instance, corev1.EventTypeNormal, "ReconcileAgentToken", "Object marked as deleted, need to delete project first")
		return r.deleteAgentToken(ctx, t)
	}

	return nil
}
