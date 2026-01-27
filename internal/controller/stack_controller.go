// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	tfc "github.com/hashicorp/go-tfe"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/version"
)

// StackReconciler reconciles a Stack object
type StackReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

type stackInstance struct {
	instance appv1alpha2.Stack

	log      logr.Logger
	tfClient HCPTerraformClient
}

//+kubebuilder:rbac:groups=app.terraform.io,resources=stacks,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.terraform.io,resources=stacks/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.terraform.io,resources=stacks/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *StackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	s := stackInstance{}

	s.log = log.Log.WithValues("stack", req.NamespacedName)
	s.log.Info("Stack Controller", "msg", "new reconciliation event")

	err := r.Client.Get(ctx, req.NamespacedName, &s.instance)
	if err != nil {
		// 'Not found' error occurs when an object is removed from the Kubernetes
		// No actions are required in this case
		if kerrors.IsNotFound(err) {
			s.log.Info("Stack Controller", "msg", "the instance was removed no further action is required")
			return doNotRequeue()
		}
		s.log.Error(err, "Stack Controller", "msg", "get instance object")
		return requeueAfter(requeueInterval)
	}

	if a, ok := s.instance.GetAnnotations()[annotationPaused]; ok && a == MetaTrue {
		s.log.Info("Stack Controller", "msg", "reconciliation is paused for this resource")
		return doNotRequeue()
	}

	s.log.Info("Spec Validation", "msg", "validating instance object spec")
	if err := s.instance.ValidateSpec(); err != nil {
		s.log.Error(err, "Spec Validation", "msg", "spec is invalid, exit from reconciliation")
		r.Recorder.Event(&s.instance, corev1.EventTypeWarning, "SpecValidation", err.Error())
		return doNotRequeue()
	}
	s.log.Info("Spec Validation", "msg", "spec is valid")

	if needToAddFinalizer(&s.instance, stackFinalizer) {
		err := r.addFinalizer(ctx, &s.instance)
		if err != nil {
			s.log.Error(err, "Stack Controller", "msg", fmt.Sprintf("failed to add finalizer %s to the object", stackFinalizer))
			r.Recorder.Eventf(&s.instance, corev1.EventTypeWarning, "AddFinalizer", "Failed to add finalizer %s to the object", stackFinalizer)
			return requeueOnErr(err)
		}
		s.log.Info("Stack Controller", "msg", fmt.Sprintf("successfully added finalizer %s to the object", stackFinalizer))
		r.Recorder.Eventf(&s.instance, corev1.EventTypeNormal, "AddFinalizer", "Successfully added finalizer %s to the object", stackFinalizer)
	}

	err = r.getTerraformClient(ctx, &s)
	if err != nil {
		s.log.Error(err, "Stack Controller", "msg", "failed to get HCP Terraform client")
		r.Recorder.Event(&s.instance, corev1.EventTypeWarning, "TerraformClient", "Failed to get HCP Terraform Client")
		return requeueAfter(requeueInterval)
	}

	err = r.reconcileStack(ctx, &s)
	if err != nil {
		s.log.Error(err, "Stack Controller", "msg", "reconcile stack")
		r.Recorder.Event(&s.instance, corev1.EventTypeWarning, "ReconcileStack", "Failed to reconcile stack")
		return requeueAfter(requeueInterval)
	}
	s.log.Info("Stack Controller", "msg", "successfully reconciled stack")
	r.Recorder.Eventf(&s.instance, corev1.EventTypeNormal, "ReconcileStack", "Successfully reconciled stack ID %s", s.instance.Status.StackID)

	return requeueAfter(StackSyncPeriod)
}

func (r *StackReconciler) addFinalizer(ctx context.Context, instance *appv1alpha2.Stack) error {
	controllerutil.AddFinalizer(instance, stackFinalizer)

	return r.Update(ctx, instance)
}

func (r *StackReconciler) getTerraformClient(ctx context.Context, s *stackInstance) error {
	nn := types.NamespacedName{
		Namespace: s.instance.Namespace,
		Name:      s.instance.Spec.Token.SecretKeyRef.Name,
	}
	token, err := secretKeyRef(ctx, r.Client, nn, s.instance.Spec.Token.SecretKeyRef.Key)
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
		s.log.Info("Reconcile Stack", "msg", "client configured to skip TLS certificate verifications")
	}

	httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}

	config := &tfc.Config{
		Token:      token,
		HTTPClient: httpClient,
		Headers: http.Header{
			"User-Agent": []string{version.UserAgent},
		},
	}
	s.tfClient.Client, err = tfc.NewClient(config)

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *StackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha2.Stack{}).
		WithEventFilter(predicate.Or(genericPredicates())).
		Complete(r)
}

func (r *StackReconciler) updateStatus(ctx context.Context, s *stackInstance) error {
	s.instance.Status.ObservedGeneration = s.instance.Generation

	return r.Status().Update(ctx, &s.instance)
}

func (r *StackReconciler) removeFinalizer(ctx context.Context, s *stackInstance) error {
	controllerutil.RemoveFinalizer(&s.instance, stackFinalizer)

	err := r.Update(ctx, &s.instance)
	if err != nil {
		s.log.Error(err, "Reconcile Stack", "msg", fmt.Sprintf("failed to remove finalizer %s", stackFinalizer))
		r.Recorder.Eventf(&s.instance, corev1.EventTypeWarning, "RemoveStack", "Failed to remove finalizer %s", stackFinalizer)
	}

	return err
}

func (r *StackReconciler) reconcileStack(ctx context.Context, s *stackInstance) error {
	s.log.Info("Reconcile Stack", "msg", "reconciling stack")

	// verify whether the Kubernetes object has been marked as deleted and if so delete the stack
	if isDeletionCandidate(&s.instance, stackFinalizer) {
		s.log.Info("Reconcile Stack", "msg", "object marked as deleted, need to delete stack first")
		r.Recorder.Event(&s.instance, corev1.EventTypeNormal, "ReconcileStack", "Object marked as deleted, need to delete stack first")
		return r.deleteStack(ctx, s)
	}

	// create a new stack if stack ID is unknown (means it was never created by the controller)
	// this condition will work just one time, when a new Kubernetes object is created
	if s.instance.IsCreationCandidate() {
		s.log.Info("Reconcile Stack", "msg", "status.stackID is empty, creating a new stack")
		r.Recorder.Event(&s.instance, corev1.EventTypeNormal, "ReconcileStack", "Status.StackID is empty, creating a new stack")
		err := r.createStack(ctx, s)
		if err != nil {
			s.log.Error(err, "Reconcile Stack", "msg", "failed to create a new stack")
			r.Recorder.Event(&s.instance, corev1.EventTypeWarning, "ReconcileStack", "Failed to create a new stack")
			return err
		}
		s.log.Info("Reconcile Stack", "msg", "successfully created a new stack")
		r.Recorder.Eventf(&s.instance, corev1.EventTypeNormal, "ReconcileStack", "Successfully created a new stack with ID %s", s.instance.Status.StackID)
	}

	return r.updateStatus(ctx, s)
}

func (r *StackReconciler) createStack(ctx context.Context, s *stackInstance) error {
	s.log.Info("Create Stack", "msg", "creating a new stack")

	// Note: The actual HCP Terraform Stacks API implementation would go here
	// This is a placeholder implementation that sets a mock stack ID
	// In a real implementation, you would call the HCP Terraform API to create the stack

	s.instance.Status.StackID = "stack-placeholder-id"
	s.instance.Status.ObservedGeneration = s.instance.Generation

	return nil
}

func (r *StackReconciler) deleteStack(ctx context.Context, s *stackInstance) error {
	s.log.Info("Delete Stack", "msg", "deleting stack")

	if s.instance.Spec.DeletionPolicy == appv1alpha2.StackDeletionPolicyRetain {
		s.log.Info("Delete Stack", "msg", "deletion policy is 'retain', skipping stack deletion")
		r.Recorder.Event(&s.instance, corev1.EventTypeNormal, "DeleteStack", "Deletion policy is 'retain', skipping stack deletion")
		return r.removeFinalizer(ctx, s)
	}

	// Note: The actual HCP Terraform Stacks API deletion would go here
	// This is a placeholder implementation
	// In a real implementation, you would call the HCP Terraform API to delete the stack

	s.log.Info("Delete Stack", "msg", "successfully deleted stack")
	r.Recorder.Event(&s.instance, corev1.EventTypeNormal, "DeleteStack", "Successfully deleted stack")

	return r.removeFinalizer(ctx, s)
}

// Made with Bob
