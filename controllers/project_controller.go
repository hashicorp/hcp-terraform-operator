// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	tfc "github.com/hashicorp/go-tfe"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

type projectInstance struct {
	instance appv1alpha2.Project

	log      logr.Logger
	tfClient TerraformCloudClient
}

//+kubebuilder:rbac:groups=app.terraforp.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.terraforp.io,resources=projects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.terraforp.io,resources=projects/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	p := projectInstance{}

	p.log = log.Log.WithValues("project", req.NamespacedName)
	p.log.Info("Project Controller", "msg", "new reconciliation event")

	err := r.Client.Get(ctx, req.NamespacedName, &p.instance)
	if err != nil {
		// 'Not found' error occurs when an object is removed from the Kubernetes
		// No actions are required in this case
		if errors.IsNotFound(err) {
			p.log.Info("Project Controller", "msg", "the instance was removed no further action is required")
			return doNotRequeue()
		}
		p.log.Error(err, "Project Controller", "msg", "get instance object")
		return requeueAfter(requeueInterval)
	}

	p.log.Info("Spec Validation", "msg", "validating instance object spec")
	if err := p.instance.ValidateSpec(); err != nil {
		p.log.Error(err, "Spec Validation", "msg", "spec is invalid, exit from reconciliation")
		r.Recorder.Event(&p.instance, corev1.EventTypeWarning, "SpecValidation", err.Error())
		return doNotRequeue()
	}
	p.log.Info("Spec Validation", "msg", "spec is valid")

	if needToAddFinalizer(&p.instance, projectFinalizer) {
		err := r.addFinalizer(ctx, &p.instance)
		if err != nil {
			p.log.Error(err, "Project Controller", "msg", fmt.Sprintf("failed to add finalizer %s to the object", projectFinalizer))
			r.Recorder.Eventf(&p.instance, corev1.EventTypeWarning, "AddFinalizer", "Failed to add finalizer %s to the object", projectFinalizer)
			return requeueOnErr(err)
		}
		p.log.Info("Project Controller", "msg", fmt.Sprintf("successfully added finalizer %s to the object", projectFinalizer))
		r.Recorder.Eventf(&p.instance, corev1.EventTypeNormal, "AddFinalizer", "Successfully added finalizer %s to the object", projectFinalizer)
	}

	err = r.getTerraformClient(ctx, &p)
	if err != nil {
		p.log.Error(err, "Project Controller", "msg", "failed to get terraform cloud client")
		r.Recorder.Event(&p.instance, corev1.EventTypeWarning, "TerraformClient", "Failed to get Terraform Client")
		return requeueAfter(requeueInterval)
	}

	err = r.reconcileProject(ctx, &p)
	if err != nil {
		p.log.Error(err, "Project Controller", "msg", "reconcile project")
		r.Recorder.Event(&p.instance, corev1.EventTypeWarning, "ReconcileProject", "Failed to reconcile project")
		return requeueAfter(requeueInterval)
	}
	p.log.Info("Project Controller", "msg", "successfully reconcilied project")
	r.Recorder.Eventf(&p.instance, corev1.EventTypeNormal, "ReconcileProject", "Successfully reconcilied project ID %s", p.instance.Status.ID)

	return doNotRequeue()
}

func (r *ProjectReconciler) addFinalizer(ctx context.Context, instance *appv1alpha2.Project) error {
	controllerutil.AddFinalizer(instance, projectFinalizer)

	return r.Update(ctx, instance)
}

func (r *ProjectReconciler) getSecret(ctx context.Context, name types.NamespacedName) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, name, secret)

	return secret, err
}

func (r *ProjectReconciler) getToken(ctx context.Context, instance *appv1alpha2.Project) (string, error) {
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

func (r *ProjectReconciler) getTerraformClient(ctx context.Context, p *projectInstance) error {
	token, err := r.getToken(ctx, &p.instance)
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
		p.log.Info("Reconcile Project", "msg", "client configured to skip TLS certificate verifications")
	}

	httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}

	config := &tfc.Config{
		Token:      token,
		HTTPClient: httpClient,
	}
	p.tfClient.Client, err = tfc.NewClient(config)

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha2.Project{}).
		WithEventFilter(handlePredicates()).
		Complete(r)
}

func (r *ProjectReconciler) updateStatus(ctx context.Context, p *projectInstance, project *tfc.Project) error {
	p.instance.Status.ObservedGeneration = p.instance.Generation
	p.instance.Status.ID = project.ID
	p.instance.Status.Name = project.Name

	return r.Status().Update(ctx, &p.instance)
}

func (r *ProjectReconciler) removeFinalizer(ctx context.Context, p *projectInstance) error {
	controllerutil.RemoveFinalizer(&p.instance, projectFinalizer)

	err := r.Update(ctx, &p.instance)
	if err != nil {
		p.log.Error(err, "Reconcile Project", "msg", fmt.Sprintf("failed to remove finazlier %s", projectFinalizer))
		r.Recorder.Eventf(&p.instance, corev1.EventTypeWarning, "RemoveProject", "Failed to remove finazlier %s", projectFinalizer)
	}

	return err
}

func needToUpdateProject(instance *appv1alpha2.Project, project *tfc.Project) bool {
	// generation changed
	if instance.Generation != instance.Status.ObservedGeneration {
		return true
	}

	// name changed
	if instance.Spec.Name != project.Name {
		return true
	}

	return false
}

func (r *ProjectReconciler) createProject(ctx context.Context, p *projectInstance) (*tfc.Project, error) {
	spec := p.instance.Spec
	options := tfc.ProjectCreateOptions{
		Name: spec.Name,
	}

	project, err := p.tfClient.Client.Projects.Create(ctx, spec.Organization, options)
	if err != nil {
		p.log.Error(err, "Reconcile Project", "msg", "failed to create a new project")
		r.Recorder.Event(&p.instance, corev1.EventTypeWarning, "ReconcileProject", "Failed to create a new project")
		return nil, err
	}

	p.instance.Status = appv1alpha2.ProjectStatus{
		ID: project.ID,
	}

	return project, nil
}

func (r *ProjectReconciler) readProject(ctx context.Context, p *projectInstance) (*tfc.Project, error) {
	return p.tfClient.Client.Projects.Read(ctx, p.instance.Status.ID)
}

func (r *ProjectReconciler) updateProject(ctx context.Context, p *projectInstance, project *tfc.Project) (*tfc.Project, error) {
	updateOptions := tfc.ProjectUpdateOptions{}
	spec := p.instance.Spec

	if project.Name != spec.Name {
		updateOptions.Name = tfc.String(spec.Name)
	}

	return p.tfClient.Client.Projects.Update(ctx, p.instance.Status.ID, updateOptions)
}

func (r *ProjectReconciler) deleteProject(ctx context.Context, p *projectInstance) error {
	// if the Kubernetes object doesn't have project ID, it means it a project was never created
	// in this case, remove the finalizer and let Kubernetes remove the object permanently
	if p.instance.Status.ID == "" {
		p.log.Info("Reconcile Project", "msg", fmt.Sprintf("status.ID is empty, remove finazlier %s", projectFinalizer))
		return r.removeFinalizer(ctx, p)
	}
	err := p.tfClient.Client.Projects.Delete(ctx, p.instance.Status.ID)
	if err != nil {
		// if project wasn't found, it means it was deleted from the TF Cloud bypass the operator
		// in this case, remove the finalizer and let Kubernetes remove the object permanently
		if err == tfc.ErrResourceNotFound {
			p.log.Info("Reconcile Project", "msg", fmt.Sprintf("Project ID %s not found, remove finazlier", p.instance.Status.ID))
			return r.removeFinalizer(ctx, p)
		}
		p.log.Error(err, "Reconcile Project", "msg", fmt.Sprintf("failed to delete Project ID %s, retry later", projectFinalizer))
		r.Recorder.Eventf(&p.instance, corev1.EventTypeWarning, "ReconcileProject", "Failed to delete Project ID %s, retry later", p.instance.Status.ID)
		return err
	}

	p.log.Info("Reconcile Project", "msg", fmt.Sprintf("project ID %s has been deleted, remove finazlier", p.instance.Status.ID))
	return r.removeFinalizer(ctx, p)
}

func (r *ProjectReconciler) reconcileProject(ctx context.Context, p *projectInstance) error {
	p.log.Info("Reconcile Project", "msg", "reconciling project")

	var project *tfc.Project
	var err error

	defer func() {
		// Update the status with the Project ID. This is useful if the reconciliation failed.
		// An example here would be the case when the project has been created successfully,
		// but further reconciliation steps failed.
		//
		// If a Project creation operation failed, we don't have a project object
		// and thus don't update the status. An example here would be the case when the project name has already been taken.
		//
		// Cannot call updateStatus method since it updated multiple fields and can break reconciliation logic.
		//
		// TODO:
		// - Use conditions(https://maelvls.dev/kubernetes-conditions/)
		// - Let Objects update their own status conditions
		// - Simplify updateStatus method in a way it could be called anytime
		if project != nil && project.ID != "" {
			p.instance.Status.ID = project.ID
			err = r.Status().Update(ctx, &p.instance)
			if err != nil {
				p.log.Error(err, "Project Controller", "msg", "update status with project ID")
				r.Recorder.Event(&p.instance, corev1.EventTypeWarning, "ReconcileProject", "Failed to update status with project ID")
			}
		}
	}()

	// verify whether the Kubernetes object has been marked as deleted and if so delete the project
	if isDeletionCandidate(&p.instance, projectFinalizer) {
		p.log.Info("Reconcile Project", "msg", "object marked as deleted, need to delete project first")
		r.Recorder.Event(&p.instance, corev1.EventTypeNormal, "ReconcileProject", "Object marked as deleted, need to delete project first")
		return r.deleteProject(ctx, p)
	}

	// create a new project if project ID is unknown(means it was never created by the controller)
	// this condition will work just one time, when a new Kubernetes object is created
	if p.instance.IsCreationCandidate() {
		p.log.Info("Reconcile Project", "msg", "status.ID is empty, creating a new project")
		r.Recorder.Event(&p.instance, corev1.EventTypeNormal, "ReconcileProject", "Status.ID is empty, creating a new project")
		_, err = r.createProject(ctx, p)
		if err != nil {
			p.log.Error(err, "Reconcile Project", "msg", "failed to create a new project")
			r.Recorder.Event(&p.instance, corev1.EventTypeWarning, "ReconcileProject", "Failed to create a new project")
			return err
		}
		p.log.Info("Reconcile Project", "msg", "successfully created a new project")
		r.Recorder.Eventf(&p.instance, corev1.EventTypeNormal, "ReconcileProject", "Successfully created a new project with ID %s", p.instance.Status.ID)
	}

	// read the Terraform Cloud project to compare it with the Kubernetes object spec
	project, err = r.readProject(ctx, p)
	if err != nil {
		// 'ResourceNotFound' means that the TF Cloud project was removed from the TF Cloud bypass the operator
		if err == tfc.ErrResourceNotFound {
			p.log.Info("Reconcile Project", "msg", "project not found, creating a new project")
			r.Recorder.Eventf(&p.instance, corev1.EventTypeWarning, "ReconcileProject", "Project ID %s not found, creating a new project", p.instance.Status.ID)
			project, err = r.createProject(ctx, p)
			if err != nil {
				p.log.Error(err, "Reconcile Project", "msg", "failed to create a new project")
				r.Recorder.Event(&p.instance, corev1.EventTypeWarning, "ReconcileProject", "Failed to create a new project")
				return err
			}
			p.log.Info("Reconcile Project", "msg", "successfully created a new project")
			r.Recorder.Eventf(&p.instance, corev1.EventTypeNormal, "ReconcileProject", "Successfully created a new project with ID %s", p.instance.Status.ID)
		} else {
			p.log.Error(err, "Reconcile Project", "msg", fmt.Sprintf("failed to read project ID %s", p.instance.Status.ID))
			r.Recorder.Eventf(&p.instance, corev1.EventTypeWarning, "ReconcileProject", "Failed to read project ID %s", p.instance.Status.ID)
			return err
		}
	}

	// update project if any changes have been made in the Kubernetes object spec or Terraform Cloud project
	if needToUpdateProject(&p.instance, project) {
		p.log.Info("Reconcile Project", "msg", fmt.Sprintf("observed and desired states are not matching, need to update project ID %s", p.instance.Status.ID))
		project, err = r.updateProject(ctx, p, project)
		if err != nil {
			p.log.Error(err, "Reconcile Project", "msg", fmt.Sprintf("failed to update project ID %s", p.instance.Status.ID))
			r.Recorder.Eventf(&p.instance, corev1.EventTypeWarning, "ReconcileProject", "Failed to update Project ID %s", p.instance.Status.ID)
			return err
		}
	} else {
		p.log.Info("Reconcile Project", "msg", fmt.Sprintf("observed and desired states are matching, no need to update Project ID %s", p.instance.Status.ID))
	}

	// Reconcile Team Access
	err = r.reconcileTeamAccess(ctx, p)
	if err != nil {
		p.log.Error(err, "Reconcile Team Access", "msg", fmt.Sprintf("failed to reconcile team access in project ID %s", p.instance.Status.ID))
		r.Recorder.Eventf(&p.instance, corev1.EventTypeWarning, "ReconcileTeamAccess", "Failed to reconcile team access in project ID %s", p.instance.Status.ID)
		return err
	}
	p.log.Info("Reconcile Team Access", "msg", "successfully reconcilied team access")
	r.Recorder.Eventf(&p.instance, corev1.EventTypeNormal, "ReconcileTeamAccess", "Reconcilied team access in project ID %s", p.instance.Status.ID)

	return r.updateStatus(ctx, p, project)
}
