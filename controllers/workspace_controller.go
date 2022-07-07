package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"

	tfc "github.com/hashicorp/go-tfe"
)

type TerraformCloudClient struct {
	Client *tfc.Client
}

// WorkspaceReconciler reconciles a Workspace object
type WorkspaceReconciler struct {
	client.Client
	log      logr.Logger
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
	tfClient TerraformCloudClient
}

//+kubebuilder:rbac:groups=app.terraform.io,resources=workspaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.terraform.io,resources=workspaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.terraform.io,resources=workspaces/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx)

	r.log.Info("Reconcile Workspace", "msg", "new reconciliation event")

	instance := &appv1alpha2.Workspace{}

	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		// 'Not found' error occurs when an object is removed from the Kubernetes
		// No actions are required in this case
		if errors.IsNotFound(err) {
			return doNotRequeue()
		}
		return requeueAfter(requeueInterval)
	}

	if needToAddFinalizer(instance) {
		err := r.addFinalizer(ctx, instance)
		if err != nil {
			r.log.Error(err, "add finalizer")
			return requeueOnErr(err)
		}
	}

	err = r.getTerraformClient(ctx, instance)
	if err != nil {
		return requeueAfter(requeueInterval)
	}

	err = r.reconcileWorkspace(ctx, instance)
	if err != nil {
		r.log.Error(err, "Reconcile workspace")
		return requeueAfter(requeueInterval)
	}

	return doNotRequeue()
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha2.Workspace{}).
		Complete(r)
}

// KUBERNETES HELPERS
func (r *WorkspaceReconciler) getSecret(ctx context.Context, objectKey types.NamespacedName) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, objectKey, secret)

	return secret, err
}

// TERRAFORM CLOUD PLATFORM CLIENT
func (r *WorkspaceReconciler) getToken(ctx context.Context, instance *appv1alpha2.Workspace) (string, error) {
	var secret *corev1.Secret

	secretName := instance.Spec.Token.SecretKeyRef.Name
	secretKey := instance.Spec.Token.SecretKeyRef.Key

	objectKey := types.NamespacedName{
		Namespace: instance.Namespace,
		Name:      secretName,
	}
	secret, err := r.getSecret(ctx, objectKey)
	if err != nil {
		r.log.Error(err, "Reconcile workspace")
		return "", err
	}

	if token, ok := secret.Data[secretKey]; ok {
		return string(token), nil
	}
	return "", fmt.Errorf("token key %s does not exist in the secret %s", secretKey, secretName)
}

func (r *WorkspaceReconciler) getTerraformClient(ctx context.Context, instance *appv1alpha2.Workspace) error {
	token, err := r.getToken(ctx, instance)
	if err != nil {
		return err
	}

	config := &tfc.Config{
		Token: token,
	}
	r.tfClient.Client, err = tfc.NewClient(config)

	return err
}

// HELPERS
func isDeletionCandidate(instance *appv1alpha2.Workspace) bool {
	return !instance.ObjectMeta.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(instance, workspaceFinalizer)
}

func isCreationCandidate(instance *appv1alpha2.Workspace) bool {
	return instance.Status.WorkspaceID == ""
}

func needToAddFinalizer(instance *appv1alpha2.Workspace) bool {
	return instance.ObjectMeta.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(instance, workspaceFinalizer)
}

func needToUpdateWorkspace(instance *appv1alpha2.Workspace, workspace *tfc.Workspace) bool {
	return instance.Generation != instance.Status.ObservedGeneration || workspace.UpdatedAt.Unix() != instance.Status.UpdateAt
}

// FINALIZERS
func (r *WorkspaceReconciler) addFinalizer(ctx context.Context, instance *appv1alpha2.Workspace) error {
	controllerutil.AddFinalizer(instance, workspaceFinalizer)

	return r.Update(ctx, instance)
}

func (r *WorkspaceReconciler) removeFinalizer(ctx context.Context, instance *appv1alpha2.Workspace) error {
	controllerutil.RemoveFinalizer(instance, workspaceFinalizer)

	return r.Update(ctx, instance)
}

// STATUS
func (r *WorkspaceReconciler) updateStatus(ctx context.Context, instance *appv1alpha2.Workspace, workspace *tfc.Workspace) error {
	instance.Status.ObservedGeneration = instance.Generation
	instance.Status.UpdateAt = workspace.UpdatedAt.Unix()
	instance.Status.WorkspaceID = workspace.ID

	return r.Status().Update(ctx, instance)
}

// WORKSPACES
func (r *WorkspaceReconciler) createWorkspace(ctx context.Context, instance *appv1alpha2.Workspace) error {
	spec := instance.Spec
	options := tfc.WorkspaceCreateOptions{
		Name: tfc.String(spec.Name),
	}

	workspace, err := r.tfClient.Client.Workspaces.Create(ctx, spec.Organization, options)
	if err != nil {
		return err
	}
	// Update status once a workspace has been successfully created
	return r.updateStatus(ctx, instance, workspace)
}

func (r *WorkspaceReconciler) readWorkspace(ctx context.Context, instance *appv1alpha2.Workspace) (*tfc.Workspace, error) {
	return r.tfClient.Client.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
}

func (r *WorkspaceReconciler) updateWorkspace(ctx context.Context, instance *appv1alpha2.Workspace, workspace *tfc.Workspace) (*tfc.Workspace, error) {
	var updateOptions tfc.WorkspaceUpdateOptions
	spec := instance.Spec

	if workspace.Name != spec.Name {
		updateOptions.Name = &spec.Name
	}

	return r.tfClient.Client.Workspaces.UpdateByID(ctx, instance.Status.WorkspaceID, updateOptions)
}

func (r *WorkspaceReconciler) deleteWorkspace(ctx context.Context, instance *appv1alpha2.Workspace) error {
	// if the Kubernetes object doesn't have workspace ID, it means it a workspace was never created
	// in this case, remove the finalizer and let Kubernetes remove the object permanently
	if instance.Status.WorkspaceID == "" {
		return r.removeFinalizer(ctx, instance)
	}
	err := r.tfClient.Client.Workspaces.DeleteByID(ctx, instance.Status.WorkspaceID)
	if err != nil {
		// if workspace wasn't found, it means it was deleted from the TF Cloud bypass the operator
		// in this case, remove the finalizer and let Kubernetes remove the object permanently
		if err == tfc.ErrResourceNotFound {
			return r.removeFinalizer(ctx, instance)
		}
		return err
	}

	return r.removeFinalizer(ctx, instance)
}

func (r *WorkspaceReconciler) reconcileWorkspace(ctx context.Context, instance *appv1alpha2.Workspace) error {
	var workspace *tfc.Workspace
	var err error

	// verify whether the Kubernetes object has been marked as deleted and if so delete the workspace
	if isDeletionCandidate(instance) {
		return r.deleteWorkspace(ctx, instance)
	}

	// create a new workspace if workspace ID is unknown(means it was never created by the controller)
	// this condition will work just one time, when a new Kubernetes object is created
	if isCreationCandidate(instance) {
		r.log.Info("Reconcile Workspace", "msg", "workspace ID is empty, creating a new workspace")
		return r.createWorkspace(ctx, instance)
	}

	// read the Terraform Cloud workspace to compare it with the Kubernetes object spec
	workspace, err = r.readWorkspace(ctx, instance)
	if err != nil {
		// 'ResourceNotFound' means that the TF Cloud workspace was removed from the TF Cloud bypass the operator
		if err == tfc.ErrResourceNotFound {
			r.log.Info("Reconcile Workspace", "msg", "workspace is not found, creating a new workspace")
			return r.createWorkspace(ctx, instance)
		} else {
			return err
		}
	}

	// update workspace if any changes have been made in the Kubernetes object spec or Terraform Cloud workspace
	if needToUpdateWorkspace(instance, workspace) {
		workspace, err = r.updateWorkspace(ctx, instance, workspace)
		if err != nil {
			return err
		}
	}
	// Update status once a workspace has been successfully updated
	return r.updateStatus(ctx, instance, workspace)
}
