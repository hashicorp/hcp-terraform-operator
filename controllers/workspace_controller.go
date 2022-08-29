package controllers

import (
	"context"
	"fmt"
	"strings"

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
//+kubebuilder:rbac:groups=app.terraform.io,resources=workspaces/events,verbs=create;patch
//+kubebuilder:rbac:groups=app.terraform.io,resources=workspaces/finalizers,verbs=update
//+kubebuilder:rbac:groups=app.terraform.io,resources=workspaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.Log.WithValues("workspace", req.NamespacedName)

	r.log.Info("Workspace Controller", "msg", "new reconciliation event")

	instance := &appv1alpha2.Workspace{}

	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		// 'Not found' error occurs when an object is removed from the Kubernetes
		// No actions are required in this case
		if errors.IsNotFound(err) {
			return doNotRequeue()
		}
		r.log.Error(err, "Workspace Controller", "msg", "get instance object")
		return requeueAfter(requeueInterval)
	}

	if needToAddFinalizer(instance) {
		err := r.addFinalizer(ctx, instance)
		if err != nil {
			r.log.Error(err, "Workspace Controller", "msg", fmt.Sprintf("failed to add finalizer %s to the object", workspaceFinalizer))
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, "AddFinalizer", "Failed to add finalizer %s to the object", workspaceFinalizer)
			return requeueOnErr(err)
		}
		r.log.Info("Workspace Controller", "msg", fmt.Sprintf("successfully added finalizer %s to the object", workspaceFinalizer))
		r.Recorder.Eventf(instance, corev1.EventTypeNormal, "AddFinalizer", "Successfully added finalizer %s to the object", workspaceFinalizer)
	}

	err = r.getTerraformClient(ctx, instance)
	if err != nil {
		r.log.Error(err, "Workspace Controller", "msg", "failed to get terraform cloud client")
		r.Recorder.Event(instance, corev1.EventTypeWarning, "TerraformClient", "Failed to get Terraform Client")
		return requeueAfter(requeueInterval)
	}

	err = r.reconcileWorkspace(ctx, instance)
	if err != nil {
		r.log.Error(err, "Workspace Controller", "msg", "reconcile workspace")
		r.Recorder.Event(instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to reconcile workspace")
		return requeueAfter(requeueInterval)
	}
	r.log.Info("Workspace Controller", "msg", "successfully reconcilied workspace")
	r.Recorder.Eventf(instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Successfully reconcilied workspace ID %s", instance.Status.WorkspaceID)

	return doNotRequeue()
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha2.Workspace{}).
		Complete(r)
}

// KUBERNETES HELPERS
func (r *WorkspaceReconciler) getConfigMap(ctx context.Context, name types.NamespacedName) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, name, cm)

	return cm, err
}

func (r *WorkspaceReconciler) getSecret(ctx context.Context, name types.NamespacedName) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, name, secret)

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
		return "", err
	}

	if token, ok := secret.Data[secretKey]; ok {
		return strings.TrimSuffix(string(token), "\n"), nil
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

// applyMethodToBool turns spec.applyMethod field into bool to align with the Workspace AutoApply field
// `spec.applyMethod: auto` is equal to `AutoApply: true`
// `spec.applyMethod: manual` is equal to `AutoApply: false`
func applyMethodToBool(applyMethod string) bool {
	return applyMethod == "auto"
}

// autoApplyToStr turns the Workspace AutoApply field into string to align with spec.applyMethod
// `AutoApply: true` is equal to `spec.applyMethod: auto`
// `AutoApply: false` is equal to `spec.applyMethod: manual`
func autoApplyToStr(autoApply bool) string {
	if autoApply {
		return "auto"
	}

	return "manual"
}

// FINALIZERS
func (r *WorkspaceReconciler) addFinalizer(ctx context.Context, instance *appv1alpha2.Workspace) error {
	controllerutil.AddFinalizer(instance, workspaceFinalizer)

	return r.Update(ctx, instance)
}

func (r *WorkspaceReconciler) removeFinalizer(ctx context.Context, instance *appv1alpha2.Workspace) error {
	controllerutil.RemoveFinalizer(instance, workspaceFinalizer)

	err := r.Update(ctx, instance)
	if err != nil {
		r.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to remove finazlier %s", workspaceFinalizer))
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "RemoveFinalizer", "Failed to remove finazlier %s", workspaceFinalizer)
	}

	return err
}

// STATUS
// TODO need to update this to update the spec with default values from TFC API
// change this function to updateObject?
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

		AutoApply:        tfc.Bool(applyMethodToBool(spec.ApplyMethod)),
		Description:      tfc.String(spec.Description),
		ExecutionMode:    tfc.String(spec.ExecutionMode),
		TerraformVersion: tfc.String(spec.TerraformVersion),
		WorkingDirectory: tfc.String(spec.WorkingDirectory),
	}

	workspace, err := r.tfClient.Client.Workspaces.Create(ctx, spec.Organization, options)
	if err != nil {
		r.log.Error(err, "Reconcile Workspace", "msg", "failed to create a new workspace")
		r.Recorder.Event(instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to create a new workspace")
		return err
	}
	r.log.Info("Reconcile Workspace", "msg", "successfully created a new workspace")
	r.Recorder.Eventf(instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Successfully created a new workspace with ID %s", workspace.ID)

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
		updateOptions.Name = tfc.String(spec.Name)
	}

	if workspace.AutoApply != applyMethodToBool(spec.ApplyMethod) {
		updateOptions.AutoApply = tfc.Bool(applyMethodToBool(spec.ApplyMethod))
	}
	if workspace.Description != spec.Description {
		updateOptions.Description = tfc.String(spec.Description)
	}
	if workspace.ExecutionMode != spec.ExecutionMode {
		updateOptions.ExecutionMode = tfc.String(spec.ExecutionMode)
	}
	if workspace.TerraformVersion != spec.TerraformVersion {
		updateOptions.TerraformVersion = tfc.String(spec.TerraformVersion)
	}
	if workspace.WorkingDirectory != spec.WorkingDirectory {
		updateOptions.WorkingDirectory = tfc.String(spec.WorkingDirectory)
	}

	return r.tfClient.Client.Workspaces.UpdateByID(ctx, instance.Status.WorkspaceID, updateOptions)
}

func (r *WorkspaceReconciler) deleteWorkspace(ctx context.Context, instance *appv1alpha2.Workspace) error {
	// if the Kubernetes object doesn't have workspace ID, it means it a workspace was never created
	// in this case, remove the finalizer and let Kubernetes remove the object permanently
	if instance.Status.WorkspaceID == "" {
		r.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("status.WorkspaceID is empty, remove finazlier %s", workspaceFinalizer))
		return r.removeFinalizer(ctx, instance)
	}
	err := r.tfClient.Client.Workspaces.DeleteByID(ctx, instance.Status.WorkspaceID)
	if err != nil {
		// if workspace wasn't found, it means it was deleted from the TF Cloud bypass the operator
		// in this case, remove the finalizer and let Kubernetes remove the object permanently
		if err == tfc.ErrResourceNotFound {
			r.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("Workspace ID %s was not fond, remove finazlier", workspaceFinalizer))
			return r.removeFinalizer(ctx, instance)
		}
		r.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to delete Workspace ID %s, retry later", workspaceFinalizer))
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to delete Workspace ID %s, retry later", instance.Status.WorkspaceID)
		return err
	}

	r.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("workspace ID %s has been deleted, remove finazlier", instance.Status.WorkspaceID))
	return r.removeFinalizer(ctx, instance)
}

func (r *WorkspaceReconciler) reconcileWorkspace(ctx context.Context, instance *appv1alpha2.Workspace) error {
	r.log.Info("Reconcile Workspace", "msg", "reconciling workspace")

	var workspace *tfc.Workspace
	var err error

	// verify whether the Kubernetes object has been marked as deleted and if so delete the workspace
	if isDeletionCandidate(instance) {
		r.log.Info("Reconcile Workspace", "msg", "object marked as deleted, need to delete workspace first")
		r.Recorder.Event(instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Object marked as deleted, need to delete workspace first")
		return r.deleteWorkspace(ctx, instance)
	}

	// create a new workspace if workspace ID is unknown(means it was never created by the controller)
	// this condition will work just one time, when a new Kubernetes object is created
	if isCreationCandidate(instance) {
		r.log.Info("Reconcile Workspace", "msg", "status.WorkspaceID is empty, creating a new workspace")
		r.Recorder.Event(instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Status.WorkspaceID is empty, creating a new workspace")
		return r.createWorkspace(ctx, instance)
	}

	// read the Terraform Cloud workspace to compare it with the Kubernetes object spec
	workspace, err = r.readWorkspace(ctx, instance)
	if err != nil {
		// 'ResourceNotFound' means that the TF Cloud workspace was removed from the TF Cloud bypass the operator
		if err == tfc.ErrResourceNotFound {
			r.log.Info("Reconcile Workspace", "msg", "workspace was not found, creating a new workspace")
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Workspace ID %s was not found, creating a new workspace", instance.Status.WorkspaceID)
			return r.createWorkspace(ctx, instance)
		} else {
			r.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to read workspace ID %s", instance.Status.WorkspaceID))
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to read workspace ID %s", instance.Status.WorkspaceID)
			return err
		}
	}

	// update workspace if any changes have been made in the Kubernetes object spec or Terraform Cloud workspace
	if needToUpdateWorkspace(instance, workspace) {
		r.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("observed and desired states are not matching, need to update workspace ID %s", instance.Status.WorkspaceID))
		workspace, err = r.updateWorkspace(ctx, instance, workspace)
		if err != nil {
			r.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to update workspace ID %s", instance.Status.WorkspaceID))
			r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to update workspace ID %s", instance.Status.WorkspaceID)
			return err
		}
	} else {
		r.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("observed and desired states are matching, no need to update workspace ID %s", instance.Status.WorkspaceID))
	}

	err = r.reconcileTags(ctx, instance, workspace)
	if err != nil {
		r.log.Error(err, "Reconcile Tags", "msg", "reconcile tags")
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileTags", "Failed to reconcile tags in workspace ID %s", instance.Status.WorkspaceID)
		return err
	}
	r.log.Info("Reconcile Tags", "msg", "successfully reconcilied tags")
	r.Recorder.Eventf(instance, corev1.EventTypeNormal, "ReconcileTags", "Successfully reconcilied tags in workspace ID %s", instance.Status.WorkspaceID)

	err = r.reconcileVariables(ctx, instance, workspace)
	if err != nil {
		r.log.Error(err, "Reconcile Variables", "msg", fmt.Sprintf("failed to reconcile variables in workspace ID %s", instance.Status.WorkspaceID))
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "ReconcileVariables", "Failed to reconcile variables in workspace ID %s", instance.Status.WorkspaceID)
		return err
	}
	r.log.Info("Reconcile Variables", "msg", "successfully reconcilied variables")
	r.Recorder.Eventf(instance, corev1.EventTypeNormal, "ReconcileVariables", "Reconcilied variables in workspace ID %s", instance.Status.WorkspaceID)

	// Update status once a workspace has been successfully updated
	return r.updateStatus(ctx, instance, workspace)
}
