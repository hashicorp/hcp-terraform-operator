// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-slug"
	tfc "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
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
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
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

	// token and httpClient are used to upload configuration versions,
	// which is not covered by the generated API client.
	token      string
	httpClient *http.Client
}

var (
	runStatusComplete = map[string]struct{}{
		models.APPLIED_RUNS_ATTRIBUTES_STATUS.String():              {},
		models.PLANNED_AND_FINISHED_RUNS_ATTRIBUTES_STATUS.String(): {},
	}

	runStatusUnsuccessful = map[string]struct{}{
		models.CANCELED_RUNS_ATTRIBUTES_STATUS.String():  {},
		models.DISCARDED_RUNS_ATTRIBUTES_STATUS.String(): {},
		models.ERRORED_RUNS_ATTRIBUTES_STATUS.String():   {},
	}
)

const (
	configurationVersionUploaded = "uploaded"
	configurationVersionErrored  = "errored"
)

// uploadConfigurationVersion packs a given directory and uploads it to a given upload URL.
// Same as go-tfe v1, rate limited (429) responses are retried.
func (m *moduleInstance) uploadConfigurationVersion(ctx context.Context, uploadURL, path string) error {
	body := bytes.NewBuffer(nil)
	if _, err := slug.Pack(path, body, true); err != nil {
		return err
	}
	payload := body.Bytes()

	// The upload URL is pre-signed (archivist). The token is sent only to the API host.
	sendToken := false
	base := m.tfClient.Client.BaseURL()
	if u, err := url.Parse(uploadURL); err == nil && u.Host == base.Host {
		sendToken = true
	}

	const maxRetries = 30
	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/octet-stream")
		if sendToken {
			req.Header.Set("Authorization", "Bearer "+m.token)
		}
		req.Header.Set("User-Agent", version.UserAgent)

		resp, err := m.httpClient.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRetries {
			_ = resp.Body.Close()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(400 * time.Millisecond):
			}
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("failed to upload configuration version: %s: %s", resp.Status, string(b))
		}
		return nil
	}
}

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
		if kerrors.IsNotFound(err) {
			m.log.Info("Module Controller", "msg", "the instance was removed no further action is required")
			return doNotRequeue()
		}
		m.log.Error(err, "Module Controller", "msg", "get instance object")
		return requeueAfter(requeueInterval)
	}

	if a, ok := m.instance.GetAnnotations()[annotationPaused]; ok && a == MetaTrue {
		m.log.Info("Module Controller", "msg", "reconciliation is paused for this resource")
		return doNotRequeue()
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

func (r *ModuleReconciler) updateStatusCV(ctx context.Context, instance *appv1alpha2.Module, workspace models.Workspacesable, cv models.ConfigurationVersionsable) error {
	instance.Status.WorkspaceID = pointer.ValueOf(workspace.GetId())
	instance.Status.ObservedGeneration = instance.Generation
	if cv != nil {
		var status string
		if s := cv.GetAttributes().GetStatus(); s != nil {
			status = s.String()
		}
		instance.Status.ConfigurationVersion = &appv1alpha2.ConfigurationVersionStatus{
			ID:     pointer.ValueOf(cv.GetId()),
			Status: status,
		}
		// Erase the run status since we proceeding with a new config version
		instance.Status.Run = nil
	}

	return r.Status().Update(ctx, instance)
}

func (r *ModuleReconciler) updateStatusRun(ctx context.Context, instance *appv1alpha2.Module, workspace models.Workspacesable, run models.Runsable) error {
	var cvID string
	if cv := run.GetRelationships().GetConfigurationVersion(); cv != nil && cv.GetData() != nil {
		cvID = pointer.ValueOf(cv.GetData().GetId())
	}
	instance.Status.WorkspaceID = pointer.ValueOf(workspace.GetId())
	instance.Status.ObservedGeneration = instance.Generation
	instance.Status.Run = &appv1alpha2.RunStatus{
		ID:                   pointer.ValueOf(run.GetId()),
		Status:               runAttributesStatus(run.GetAttributes()),
		ConfigurationVersion: cvID,
	}

	return r.Status().Update(ctx, instance)
}

func (r *ModuleReconciler) updateStatusOutputs(ctx context.Context, instance *appv1alpha2.Module, workspace models.Workspacesable) error {
	instance.Status.WorkspaceID = pointer.ValueOf(workspace.GetId())
	instance.Status.ObservedGeneration = instance.Generation

	return r.Status().Update(ctx, instance)
}

func (r *ModuleReconciler) updateStatusDestroy(ctx context.Context, instance *appv1alpha2.Module, run models.Runsable) error {
	var cvID string
	if cv := run.GetRelationships().GetConfigurationVersion(); cv != nil && cv.GetData() != nil {
		cvID = pointer.ValueOf(cv.GetData().GetId())
	}
	instance.Status.DestroyRunID = pointer.ValueOf(run.GetId())
	instance.Status.ObservedGeneration = instance.Generation
	instance.Status.Run = &appv1alpha2.RunStatus{
		ID:                   pointer.ValueOf(run.GetId()),
		Status:               runAttributesStatus(run.GetAttributes()),
		ConfigurationVersion: cvID,
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

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}

	config := &tfc.Config{
		Address:           os.Getenv("TFE_ADDRESS"),
		Token:             token,
		HTTPTransport:     newRunStatusTransport(transport),
		RetryRateLimited:  true,
		RetryServerErrors: false,
		RetryMaxRetries:   30,
		Headers: http.Header{
			"User-Agent": []string{version.UserAgent},
		},
	}
	m.tfClient.Client, err = tfc.NewClient(config)

	// The configuration version upload is not covered by the API client
	m.token = token
	m.httpClient = &http.Client{Transport: transport}

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
	case configurationVersionUploaded:
		return false
	case configurationVersionErrored:
		return false

	}
	return true
}

// needNewRun checks is a new Run is required
func needNewRun(instance *appv1alpha2.Module) bool {
	if instance.Status.ConfigurationVersion == nil {
		return false
	}

	if instance.Status.ConfigurationVersion.Status == configurationVersionErrored {
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
	m.log.Info("Reconcile Module Workspace", "msg", fmt.Sprintf("successfully got workspace ID %s", pointer.ValueOf(workspace.GetId())))

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
		cvAttrs := models.NewConfigurationVersions_attributes()
		cvAttrs.SetAutoQueueRuns(new(false))
		cvData := models.NewConfigurationVersions()
		cvData.SetTypeEscaped(new(models.CONFIGURATIONVERSIONS_CONFIGURATIONVERSIONS_TYPE))
		cvData.SetAttributes(cvAttrs)
		cvBody := models.NewConfigurationVersionsEnvelope()
		cvBody.SetData(cvData)
		cvResp, err := m.tfClient.Client.API.Workspaces().ByWorkspace_id(pointer.ValueOf(workspace.GetId())).ConfigurationVersions().Post(ctx, cvBody, nil)
		if err != nil {
			m.log.Error(err, "Reconcile Configuration Version", "msg", "failed to create a new configuration versions")
			return err
		}
		cv := cvResp.GetData()
		m.log.Info("Reconcile Configuration Version", "msg", "successfully created new config version")

		m.log.Info("Reconcile Configuration Version", "msg", "upload a new config version")
		err = m.uploadConfigurationVersion(ctx, pointer.ValueOf(cv.GetAttributes().GetUploadUrl()), path)
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
		cvResp, err := m.tfClient.Client.API.ConfigurationVersions().ByConfiguration_version_id(m.instance.Status.ConfigurationVersion.ID).Get(ctx, nil)
		if err != nil {
			m.log.Error(err, "Reconcile Configuration Version", "msg", "failed to get the upload status")
			return err
		}
		cv := cvResp.GetData()
		m.log.Info("Reconcile Configuration Version", "msg", fmt.Sprintf("successfully got the upload status: %s", cv.GetAttributes().GetStatus()))
		if err := r.updateStatusCV(ctx, &m.instance, workspace, cv); err != nil {
			return err
		}
	}

	// checks if a new Run needs to be initialized
	if needNewRun(&m.instance) {
		m.log.Info("Reconcile Run", "msg", "create a new run")
		attrs := models.NewRuns_attributes()
		attrs.SetMessage(new(runMessage))
		wsData := models.NewWorkspacesHasOne_data()
		wsData.SetId(workspace.GetId())
		wsData.SetTypeEscaped(new(models.WORKSPACES_WORKSPACESIDENTIFIER_TYPE))
		wsRel := models.NewWorkspacesHasOne()
		wsRel.SetData(wsData)
		cvData := models.NewConfigurationVersionsHasOne_data()
		cvData.SetId(new(m.instance.Status.ConfigurationVersion.ID))
		cvData.SetTypeEscaped(new(models.CONFIGURATIONVERSIONS_CONFIGURATIONVERSIONSIDENTIFIER_TYPE))
		cvRel := models.NewConfigurationVersionsHasOne()
		cvRel.SetData(cvData)
		rels := models.NewRuns_relationships()
		rels.SetWorkspace(wsRel)
		rels.SetConfigurationVersion(cvRel)
		data := models.NewRuns()
		data.SetTypeEscaped(new(models.RUNS_RUNS_TYPE))
		data.SetAttributes(attrs)
		data.SetRelationships(rels)
		body := models.NewRunsEnvelope()
		body.SetData(data)
		resp, err := m.tfClient.Client.API.Runs().Post(ctx, body, nil)
		if err != nil {
			m.log.Error(err, "Reconcile Run", "msg", "failed to create a new run")
			return err
		}
		run := resp.GetData()
		m.log.Info("Reconcile Run", "msg", "successfully created a new run")

		// It can take a while to proceed with a new run
		// To unblock a worker we return the object back to the queue
		// and validate the run status during the next reconciliation
		return r.updateStatusRun(ctx, &m.instance, workspace, run)
	}

	// checks if a new version of the Run is finished
	if waitRunToComplete(m.instance.Status.Run) {
		m.log.Info("Reconcile Run", "msg", "check the run status")
		resp, err := m.tfClient.Client.API.Runs().ById(m.instance.Status.Run.ID).Get(ctx, nil)
		if err != nil {
			m.log.Error(err, "Reconcile Run", "msg", "failed to get run status")
			return err
		}
		run := resp.GetData()
		m.log.Info("Reconcile Run", "msg", fmt.Sprintf("successfully got the run status: %s", runAttributesStatus(run.GetAttributes())))
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
