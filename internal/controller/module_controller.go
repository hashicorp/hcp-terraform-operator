// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"text/template"

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
	"github.com/hashicorp/go-slug"
	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/version"
)

// ModuleReconciler reconciles a Module object
type ModuleReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

type moduleInstance struct {
	instance appv1alpha2.Module

	log      logr.Logger
	tfClient HCPTerraformClient
}

var (
	runStatusComplete = map[tfc.RunStatus]struct{}{
		tfc.RunApplied:            {},
		tfc.RunPlannedAndFinished: {},
	}

	runStatusUnsuccessful = map[tfc.RunStatus]struct{}{
		tfc.RunCanceled:  {},
		tfc.RunDiscarded: {},
		tfc.RunErrored:   {},
	}
)

// +kubebuilder:rbac:groups=app.terraform.io,resources=modules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.terraform.io,resources=modules/finalizers,verbs=update
// +kubebuilder:rbac:groups=app.terraform.io,resources=modules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;list;update;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;list;update;watch

func (r *ModuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	m := moduleInstance{}

	m.log = log.Log.WithValues("module", req.NamespacedName)
	m.log.Info("Module Controller", "msg", "new reconciliation event")

	err := r.Client.Get(ctx, req.NamespacedName, &m.instance)
	if err != nil {
		// 'Not found' error occurs when an object is removed from the Kubernetes
		// No actions are required in this case
		if errors.IsNotFound(err) {
			m.log.Info("Module Controller", "msg", "the instance was removed no further action is required")
			return doNotRequeue()
		}
		m.log.Error(err, "Module Controller", "msg", "get instance object")
		return requeueAfter(requeueInterval)
	}

	m.log.Info("Spec Validation", "msg", "validating instance object spec")
	if err := m.instance.ValidateSpec(); err != nil {
		m.log.Error(err, "Spec Validation", "msg", "spec is invalid, exit from reconciliation")
		r.Recorder.Event(&m.instance, corev1.EventTypeWarning, "SpecValidation", err.Error())
		return doNotRequeue()
	}
	m.log.Info("Spec Validation", "msg", "spec is valid")

	if needToAddFinalizer(&m.instance, moduleFinalizer) {
		err := r.addFinalizer(ctx, &m.instance)
		if err != nil {
			m.log.Error(err, "Module Controller", "msg", fmt.Sprintf("failed to add finalizer %s to the object", moduleFinalizer))
			r.Recorder.Eventf(&m.instance, corev1.EventTypeWarning, "AddFinalizer", "Failed to add finalizer %s to the object", moduleFinalizer)
			return requeueOnErr(err)
		}
		m.log.Info("Module Controller", "msg", fmt.Sprintf("successfully added finalizer %s to the object", moduleFinalizer))
		r.Recorder.Eventf(&m.instance, corev1.EventTypeNormal, "AddFinalizer", "Successfully added finalizer %s to the object", moduleFinalizer)
	}

	err = r.getTerraformClient(ctx, &m)
	if err != nil {
		m.log.Error(err, "Module Controller", "msg", "failed to get HCP Terraform client")
		r.Recorder.Event(&m.instance, corev1.EventTypeWarning, "TerraformClient", "Failed to get HCP Terraform Client")
		return requeueAfter(requeueInterval)
	}

	err = r.reconcileModule(ctx, &m)
	if err != nil {
		m.log.Error(err, "Module Controller", "msg", "reconcile module")
		r.Recorder.Event(&m.instance, corev1.EventTypeWarning, "ReconcileModule", "Failed to reconcile module")
		return requeueAfter(requeueInterval)
	}

	if waitForUploadModule(&m.instance) {
		m.log.Info("Module Controller", "msg", "waiting for configuration version to be uploaded")
		return requeueAfter(requeueConfigurationUploadInterval)
	}

	if needNewRun(&m.instance) {
		m.log.Info("Module Controller", "msg", "new config version is available, need a new run")
		return requeueAfter(requeueNewRunInterval)
	}

	if waitRunToComplete(m.instance.Status.Run) {
		m.log.Info("Module Controller", "msg", "waiting for run to finish")
		return requeueAfter(requeueRunStatusInterval)
	}

	m.log.Info("Module Controller", "msg", "successfully reconcilied module")

	return requeueAfter(ModuleSyncPeriod)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ModuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha2.Module{}).
		WithEventFilter(predicate.Or(genericPredicates())).
		Complete(r)
}

func (r *ModuleReconciler) updateStatusCV(ctx context.Context, instance *appv1alpha2.Module, workspace *tfc.Workspace, cv *tfc.ConfigurationVersion) error {
	instance.Status.WorkspaceID = workspace.ID
	instance.Status.ObservedGeneration = instance.Generation
	if cv != nil {
		instance.Status.ConfigurationVersion = &appv1alpha2.ConfigurationVersionStatus{
			ID:     cv.ID,
			Status: string(cv.Status),
		}
		// Erase the run status since we proceeding with a new config version
		instance.Status.Run = nil
	}

	return r.Status().Update(ctx, instance)
}

func (r *ModuleReconciler) updateStatusRun(ctx context.Context, instance *appv1alpha2.Module, workspace *tfc.Workspace, run *tfc.Run) error {
	instance.Status.WorkspaceID = workspace.ID
	instance.Status.ObservedGeneration = instance.Generation
	instance.Status.Run = &appv1alpha2.RunStatus{
		ID:                   run.ID,
		Status:               string(run.Status),
		ConfigurationVersion: run.ConfigurationVersion.ID,
	}

	return r.Status().Update(ctx, instance)
}

func (r *ModuleReconciler) updateStatusOutputs(ctx context.Context, instance *appv1alpha2.Module, workspace *tfc.Workspace) error {
	instance.Status.WorkspaceID = workspace.ID
	instance.Status.ObservedGeneration = instance.Generation

	return r.Status().Update(ctx, instance)
}

func (r *ModuleReconciler) updateStatusDestroy(ctx context.Context, instance *appv1alpha2.Module, run *tfc.Run) error {
	instance.Status.DestroyRunID = run.ID
	instance.Status.ObservedGeneration = instance.Generation
	instance.Status.Run = &appv1alpha2.RunStatus{
		ID:                   run.ID,
		Status:               string(run.Status),
		ConfigurationVersion: run.ConfigurationVersion.ID,
	}

	return r.Status().Update(ctx, instance)
}

func (r *ModuleReconciler) getTerraformClient(ctx context.Context, m *moduleInstance) error {
	nn := types.NamespacedName{
		Namespace: m.instance.Namespace,
		Name:      m.instance.Spec.Token.SecretKeyRef.Name,
	}
	token, err := secretKeyRef(ctx, r.Client, nn, m.instance.Spec.Token.SecretKeyRef.Key)
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
		m.log.Info("Reconcile Module", "msg", "client configured to skip TLS certificate verifications")
	}

	httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}

	config := &tfc.Config{
		Token:      token,
		HTTPClient: httpClient,
		Headers: http.Header{
			"User-Agent": []string{version.UserAgent},
		},
	}
	m.tfClient.Client, err = tfc.NewClient(config)

	return err
}

func (r *ModuleReconciler) removeFinalizer(ctx context.Context, m *moduleInstance) error {
	controllerutil.RemoveFinalizer(&m.instance, moduleFinalizer)

	err := r.Update(ctx, &m.instance)
	if err != nil {
		m.log.Error(err, "Reconcile Module", "msg", fmt.Sprintf("failed to remove finalizer %s", moduleFinalizer))
		r.Recorder.Eventf(&m.instance, corev1.EventTypeWarning, "RemoveFinalizer", "Failed to remove finalizer %s", moduleFinalizer)
	}

	return err
}

func (r *ModuleReconciler) addFinalizer(ctx context.Context, instance *appv1alpha2.Module) error {
	controllerutil.AddFinalizer(instance, moduleFinalizer)

	return r.Update(ctx, instance)
}

func generateModule(spec *appv1alpha2.ModuleSpec) (string, error) {
	td, err := os.MkdirTemp("", "tf-*")
	if err != nil {
		return td, err
	}

	f, err := os.CreateTemp(td, "*.tf")
	if err != nil {
		return td, err
	}

	t, err := template.New("module").Parse(moduleTemplate)
	if err != nil {
		return td, err
	}
	err = t.Execute(f, spec)
	if err != nil {
		return td, err
	}

	b := bytes.NewBuffer(nil)
	_, err = slug.Pack(td, b, false)
	if err != nil {
		return td, err
	}

	return td, nil
}

func needToUploadModule(instance *appv1alpha2.Module) bool {
	return instance.Generation != instance.Status.ObservedGeneration
}

// waitForUploadModule checks if need to wait for CV upload to finish
func waitForUploadModule(instance *appv1alpha2.Module) bool {
	if instance.Status.ConfigurationVersion == nil {
		return false
	}

	switch instance.Status.ConfigurationVersion.Status {
	case string(tfc.ConfigurationUploaded):
		return false
	case string(tfc.ConfigurationErrored):
		return false

	}
	return true
}

// needNewRun checks is a new Run is required
func needNewRun(instance *appv1alpha2.Module) bool {
	if instance.Status.ConfigurationVersion == nil {
		return false
	}

	if instance.Status.ConfigurationVersion.Status == string(tfc.ConfigurationErrored) {
		return false
	}

	if instance.Status.Run == nil {
		return true
	}

	if instance.Status.Run.ConfigurationVersion != instance.Status.ConfigurationVersion.ID {
		return true
	}

	return false
}

// waitRunToComplete validates whether need to wait for the current Run to finish.
func waitRunToComplete(runStatus *appv1alpha2.RunStatus) bool {
	// In the current Run status is not available yet, there is nothing to wait for.
	if runStatus == nil {
		return false
	}

	// Wait if the current Run is not completed.
	return !runStatus.RunCompleted()
}

func (r *ModuleReconciler) reconcileModule(ctx context.Context, m *moduleInstance) error {
	m.log.Info("Reconcile Module", "msg", "reconciling module")

	// verify whether the Kubernetes object has been marked as deleted and if so delete the module
	if isDeletionCandidate(&m.instance, moduleFinalizer) {
		m.log.Info("Reconcile Module", "msg", "object marked as deleted")
		r.Recorder.Event(&m.instance, corev1.EventTypeNormal, "ReconcileModule", "Object marked as deleted")
		return r.deleteModule(ctx, m)
	}

	workspace, err := r.getWorkspace(ctx, m)
	if err != nil {
		m.log.Info("Reconcile Module Workspace", "msg", "failed to get workspace")
		r.Recorder.Event(&m.instance, corev1.EventTypeWarning, "ReconcileModule", "Failed to get workspace")
		return err
	}
	m.log.Info("Reconcile Module Workspace", "msg", fmt.Sprintf("successfully got workspace ID %s", workspace.ID))

	// checks if a new version of the CV needs to be uploaded
	if needToUploadModule(&m.instance) {
		m.log.Info("Reconcile Configuration Version", "msg", "generate a new module code")
		path, err := generateModule(&m.instance.Spec)
		defer os.RemoveAll(path)
		if err != nil {
			m.log.Error(err, "Reconcile Configuration Version", "msg", "failed to generate a new module code")
			return err
		}
		m.log.Info("Reconcile Configuration Version", "msg", "successfully generated a new module code")

		m.log.Info("Reconcile Configuration Version", "msg", "create a new configuration versions")
		cv, err := m.tfClient.Client.ConfigurationVersions.Create(ctx, workspace.ID, tfc.ConfigurationVersionCreateOptions{
			AutoQueueRuns: tfc.Bool(false),
		})
		if err != nil {
			m.log.Error(err, "Reconcile Configuration Version", "msg", "failed to create a new configuration versions")
			return err
		}
		m.log.Info("Reconcile Configuration Version", "msg", "successfully created new config version")

		m.log.Info("Reconcile Configuration Version", "msg", "upload a new config version")
		err = m.tfClient.Client.ConfigurationVersions.Upload(ctx, cv.UploadURL, path)
		if err != nil {
			m.log.Error(err, "Reconcile Configuration Version", "msg", "failed to upload a new config version")
			return err
		}
		m.log.Info("Reconcile Configuration Version", "msg", "successfully uploaded a new config version")

		// It can take a few seconds to proceed with a new upload
		// To unblock a worker we return the object back to the queue
		// and validate the upload status during the next reconciliation
		return r.updateStatusCV(ctx, &m.instance, workspace, cv)
	}

	// checks if a new version of the CV is uploaded
	if waitForUploadModule(&m.instance) {
		m.log.Info("Reconcile Configuration Version", "msg", "check the upload status")
		cv, err := m.tfClient.Client.ConfigurationVersions.Read(ctx, m.instance.Status.ConfigurationVersion.ID)
		if err != nil {
			m.log.Error(err, "Reconcile Configuration Version", "msg", "failed to get the upload status")
			return err
		}
		m.log.Info("Reconcile Configuration Version", "msg", fmt.Sprintf("successfully got the upload status: %s", cv.Status))
		if err := r.updateStatusCV(ctx, &m.instance, workspace, cv); err != nil {
			return err
		}
	}

	// checks if a new Run needs to be initialized
	if needNewRun(&m.instance) {
		m.log.Info("Reconcile Run", "msg", "create a new run")
		run, err := m.tfClient.Client.Runs.Create(ctx, tfc.RunCreateOptions{
			Message:   tfc.String(runMessage),
			Workspace: workspace,
		})
		if err != nil {
			m.log.Error(err, "Reconcile Run", "msg", "failed to create a new run")
			return err
		}
		m.log.Info("Reconcile Run", "msg", "successfully created a new run")

		// It can take a while to proceed with a new run
		// To unblock a worker we return the object back to the queue
		// and validate the run status during the next reconciliation
		return r.updateStatusRun(ctx, &m.instance, workspace, run)
	}

	// checks if a new version of the Run is finished
	if waitRunToComplete(m.instance.Status.Run) {
		m.log.Info("Reconcile Run", "msg", "check the run status")
		run, err := m.tfClient.Client.Runs.Read(ctx, m.instance.Status.Run.ID)
		if err != nil {
			m.log.Error(err, "Reconcile Run", "msg", "failed to get run status")
			return err
		}
		m.log.Info("Reconcile Run", "msg", fmt.Sprintf("successfully got the run status: %s", run.Status))
		if err := r.updateStatusRun(ctx, &m.instance, workspace, run); err != nil {
			return err
		}
	}

	// Reconcile Outputs
	err = r.reconcileOutputs(ctx, m, workspace)
	if err != nil {
		m.log.Error(err, "Reconcile Module Outputs", "msg", "failed to reconcile outputs")
		r.Recorder.Event(&m.instance, corev1.EventTypeWarning, "ReconcileModuleOutputs", "Failed to reconcile outputs")
		return err
	}
	m.log.Info("Reconcile Module Outputs", "msg", "successfully reconcilied outputs")
	r.Recorder.Event(&m.instance, corev1.EventTypeNormal, "ReconcileModuleOutputs", "Successfully reconcilied outputs")

	return r.updateStatusOutputs(ctx, &m.instance, workspace)
}
