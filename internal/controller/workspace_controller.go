// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	tfc "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	abstractions "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoft/kiota-abstractions-go/serialization"
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

type HCPTerraformClient struct {
	Client *tfc.Client
}

// WorkspaceReconciler reconciles a Workspace object
type WorkspaceReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

type workspaceInstance struct {
	instance appv1alpha2.Workspace

	log      logr.Logger
	tfClient HCPTerraformClient
}

// +kubebuilder:rbac:groups=app.terraform.io,resources=workspaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.terraform.io,resources=workspaces/finalizers,verbs=update
// +kubebuilder:rbac:groups=app.terraform.io,resources=workspaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;list;update;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;list;update;watch

func (r *WorkspaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	w := workspaceInstance{}

	w.log = log.Log.WithValues("workspace", req.NamespacedName)
	w.log.Info("Workspace Controller", "msg", "new reconciliation event")

	err := r.Client.Get(ctx, req.NamespacedName, &w.instance)
	if err != nil {
		// 'Not found' error occurs when an object is removed from the Kubernetes
		// No actions are required in this case
		if kerrors.IsNotFound(err) {
			w.log.Info("Workspace Controller", "msg", "the object is removed no further action is required")
			return doNotRequeue()
		}
		w.log.Error(err, "Workspace Controller", "msg", "get instance object")
		return requeueAfter(requeueInterval)
	}

	// TODO:
	// - Think about using the DeleteFunc predicate.
	if w.instance.DeletionTimestamp != nil && !controllerutil.ContainsFinalizer(&w.instance, workspaceFinalizer) {
		w.log.Info("Workspace Controller", "msg", "object marked as deleted without finalizer, no further action is required")
		return doNotRequeue()
	}

	if a, ok := w.instance.GetAnnotations()[annotationPaused]; ok && a == MetaTrue {
		w.log.Info("Workspace Controller", "msg", "reconciliation is paused for this resource")
		return doNotRequeue()
	}

	// Migration Validation
	if controllerutil.ContainsFinalizer(&w.instance, workspaceFinalizerAlpha1) {
		w.log.Error(err, "Migration", "msg", fmt.Sprintf("spec contains old finalizer %s", workspaceFinalizerAlpha1))
		return doNotRequeue()
	}

	w.log.Info("Spec Validation", "msg", "validating instance object spec")
	if err := w.instance.ValidateSpec(); err != nil {
		w.log.Error(err, "Spec Validation", "msg", "spec is invalid, exit from reconciliation")
		r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "SpecValidation", err.Error())
		return doNotRequeue()
	}
	w.log.Info("Spec Validation", "msg", "spec is valid")

	if needToAddFinalizer(&w.instance, workspaceFinalizer) {
		err := r.addFinalizer(ctx, &w.instance)
		if err != nil {
			w.log.Error(err, "Workspace Controller", "msg", fmt.Sprintf("failed to add finalizer %s to the object", workspaceFinalizer))
			r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "AddFinalizer", "Failed to add finalizer %s to the object", workspaceFinalizer)
			return requeueOnErr(err)
		}
		w.log.Info("Workspace Controller", "msg", fmt.Sprintf("successfully added finalizer %s to the object", workspaceFinalizer))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "AddFinalizer", "Successfully added finalizer %s to the object", workspaceFinalizer)
	}

	err = r.getTerraformClient(ctx, &w)
	if err != nil {
		w.log.Error(err, "Workspace Controller", "msg", "failed to get HCP Terraform client")
		r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "TerraformClient", "Failed to get HCP Terraform Client")
		return requeueAfter(requeueInterval)
	}

	err = r.reconcileWorkspace(ctx, &w)
	if err != nil {
		w.log.Error(err, "Workspace Controller", "msg", "reconcile workspace")
		r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to reconcile workspace")
		return requeueAfter(requeueInterval)
	}
	w.log.Info("Workspace Controller", "msg", "successfully reconcilied workspace")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Successfully reconcilied workspace ID %s", w.instance.Status.WorkspaceID)

	if w.instance.Status.Run != nil && !w.instance.Status.Run.RunCompleted() {
		w.log.Info("Workspace Controller", "msg", fmt.Sprintf("current run %s status %s is not completed need to requeue", w.instance.Status.Run.ID, w.instance.Status.Run.Status))
		return requeueAfter(requeueRunStatusInterval)
	}

	if w.instance.Status.Plan != nil && !w.instance.Status.Plan.RunCompleted() {
		w.log.Info("Workspace Controller", "msg", fmt.Sprintf("speculative plan run %s status %s is not completed need to requeue", w.instance.Status.Plan.ID, w.instance.Status.Plan.Status))
		return requeueAfter(requeueRunStatusInterval)
	}

	return requeueAfter(WorkspaceSyncPeriod)
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkspaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha2.Workspace{}).
		WithEventFilter(predicate.Or(genericPredicates(), workspacePredicates())).
		Complete(r)
}

func (r *WorkspaceReconciler) getTerraformClient(ctx context.Context, w *workspaceInstance) error {
	nn := types.NamespacedName{
		Namespace: w.instance.Namespace,
		Name:      w.instance.Spec.Token.SecretKeyRef.Name,
	}
	token, err := secretKeyRef(ctx, r.Client, nn, w.instance.Spec.Token.SecretKeyRef.Key)
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
		w.log.Info("Reconcile Workspace", "msg", "client configured to skip TLS certificate verifications")
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
	w.tfClient.Client, err = tfc.NewClient(config)

	return err
}

// attributesWithAgentPool adds the 'agent-pool-id' attribute to the serialized workspace attributes.
// go-tfe v1 set the agent pool of a workspace this way, and the generated attributes model does not have this field.
type attributesWithAgentPool struct {
	models.Workspaces_attributesable
	agentPoolID string
}

// Serialize writes the attributes and the 'agent-pool-id' attribute.
func (a *attributesWithAgentPool) Serialize(writer serialization.SerializationWriter) error {
	if err := a.Workspaces_attributesable.Serialize(writer); err != nil {
		return err
	}
	return writer.WriteStringValue("agent-pool-id", &a.agentPoolID)
}

// withAgentPoolID returns the attributes with the 'agent-pool-id' attribute, or the attributes as is when the ID is nil.
func withAgentPoolID(attrs models.Workspaces_attributesable, id *string) models.Workspaces_attributesable {
	if id == nil {
		return attrs
	}
	return &attributesWithAgentPool{Workspaces_attributesable: attrs, agentPoolID: *id}
}

// sendRawJSON sends a request with a hand-made JSON:API payload and returns the workspace envelope from the response.
// The generated models cannot express explicit JSON nulls (for example, 'vcs-repo' or an SSH key ID set to null),
// which the HCP Terraform API requires to detach or unassign a related object.
func sendRawJSON(ctx context.Context, adapter abstractions.RequestAdapter, ri *abstractions.RequestInformation, payload any) (models.WorkspacesEnvelopeable, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ri.SetStreamContentAndContentType(b, "application/vnd.api+json")
	res, err := adapter.Send(ctx, ri, models.CreateWorkspacesEnvelopeFromDiscriminatorValue, nil)
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	env, _ := res.(models.WorkspacesEnvelopeable)
	return env, nil
}

func needToUpdateWorkspace(instance *appv1alpha2.Workspace, workspace models.Workspacesable) bool {
	attrs := workspace.GetAttributes()
	// generation changed
	if instance.Generation != instance.Status.ObservedGeneration {
		return true
	}
	// timestamp changed
	if pointer.ValueOf(attrs.GetUpdatedAt()).Unix() != instance.Status.UpdateAt {
		return true
	}
	// agent pool added
	if ap := workspace.GetRelationships().GetAgentPool(); instance.Spec.AgentPool != nil && (ap == nil || ap.GetData() == nil || pointer.ValueOf(ap.GetData().GetId()) == "") {
		return true
	}
	// The TFC API VCS behavior is inconsistent:
	//   - once a VCS is attached to a Workspace, the 'UpdateAt' attribute is updated
	//   - once attached to a Workspace VCS is updated, for instance, change a branch name, the 'UpdateAt' attribute is updated
	//   - once a VCS is detached from a Workspace, the 'UpdateAt' attribute is not updated, because of that we have to have this condition here
	if instance.Spec.VersionControl != nil && attrs.GetVcsRepo() == nil {
		return true
	}
	// Terraform version
	if instance.Spec.TerraformVersion == "" {
		if instance.Status.TerraformVersion != pointer.ValueOf(attrs.GetTerraformVersion()) {
			return true
		}
	}

	return false
}

// ApplyMethodToBool turns spec.applyMethod field into bool to align with the Workspace AutoApply field
// `spec.applyMethod: auto` is equal to `AutoApply: true`
// `spec.applyMethod: manual` is equal to `AutoApply: false`
func ApplyMethodToBool(applyMethod string) bool {
	return applyMethod == "auto"
}

// autoApplyRunTriggerToBool turns spec.autoapplyRunTrigger field into bool
func ApplyRunTriggerToBool(applyRunTrigger string) bool {
	return applyRunTrigger == "auto"
}

func (r *WorkspaceReconciler) addFinalizer(ctx context.Context, instance *appv1alpha2.Workspace) error {
	patch := client.MergeFrom(instance.DeepCopy())
	controllerutil.AddFinalizer(instance, workspaceFinalizer)

	return r.Patch(ctx, instance, patch)
}

func (r *WorkspaceReconciler) removeFinalizer(ctx context.Context, w *workspaceInstance) error {
	controllerutil.RemoveFinalizer(&w.instance, workspaceFinalizer)

	err := r.Update(ctx, &w.instance)
	if err != nil {
		w.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to remove finalizer %s", workspaceFinalizer))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "RemoveFinalizer", "Failed to remove finalizer %s", workspaceFinalizer)
	}

	return err
}

// STATUS
// TODO need to update this to update the spec with default values from TFC API
// change this function to updateObject?
func (r *WorkspaceReconciler) updateStatus(ctx context.Context, w *workspaceInstance, workspace models.Workspacesable) error {
	attrs := workspace.GetAttributes()
	w.instance.Status.ObservedGeneration = w.instance.Generation
	w.instance.Status.UpdateAt = pointer.ValueOf(attrs.GetUpdatedAt()).Unix()
	w.instance.Status.TerraformVersion = pointer.ValueOf(attrs.GetTerraformVersion())

	return r.Status().Update(ctx, &w.instance)
}

// WORKSPACES

// readOrganizationDefaultProjectID reads the organization and returns the ID of its default project.
// An empty string is returned when the organization does not have the default project.
func (w *workspaceInstance) readOrganizationDefaultProjectID(ctx context.Context) (string, error) {
	org, err := w.tfClient.Client.API.Organizations().ByOrganization_name(w.instance.Spec.Organization).Get(ctx, nil)
	if err != nil {
		return "", err
	}
	if org == nil || org.GetData() == nil || org.GetData().GetRelationships() == nil {
		return "", nil
	}
	if p := org.GetData().GetRelationships().GetDefaultProject(); p != nil && p.GetData() != nil {
		return pointer.ValueOf(p.GetData().GetId()), nil
	}
	return "", nil
}

func (r *WorkspaceReconciler) createWorkspace(ctx context.Context, w *workspaceInstance) (models.Workspacesable, error) {
	spec := w.instance.Spec
	attrs := models.NewWorkspaces_attributes()
	attrs.SetName(new(spec.Name))
	attrs.SetAllowDestroyPlan(new(spec.AllowDestroyPlan))
	attrs.SetAutoApply(new(ApplyMethodToBool(spec.ApplyMethod)))
	attrs.SetAutoApplyRunTrigger(new(ApplyRunTriggerToBool(spec.ApplyRunTrigger)))
	attrs.SetDescription(new(spec.Description))
	if mode, err := models.ParseWorkspaces_attributes_executionMode(spec.ExecutionMode); err == nil && mode != nil {
		attrs.SetExecutionMode(mode.(*models.Workspaces_attributes_executionMode))
	}
	attrs.SetGlobalRemoteState(new(false))
	attrs.SetTerraformVersion(new(spec.TerraformVersion))
	attrs.SetWorkingDirectory(new(spec.WorkingDirectory))

	rels := models.NewWorkspaces_relationships()
	var agentPoolAttr *string

	if spec.ExecutionMode == "agent" {
		agentPoolID, err := r.getAgentPoolID(ctx, w)
		if err != nil {
			w.log.Error(err, "Reconcile Workspace", "msg", "failed to get agent pool ID")
			r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to get agent pool ID")
			return nil, err
		}
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("agent pool ID %s will be used", agentPoolID))
		agentPoolAttr = new(agentPoolID)
	}

	if spec.VersionControl != nil {
		vcs := models.NewWorkspaces_attributes_vcsRepo()
		vcs.SetOauthTokenId(new(spec.VersionControl.OAuthTokenID))
		vcs.SetIdentifier(new(spec.VersionControl.Repository))
		vcs.SetBranch(new(spec.VersionControl.Branch))
		attrs.SetVcsRepo(vcs)
		attrs.SetSpeculativeEnabled(new(spec.VersionControl.SpeculativePlans))
		attrs.SetFileTriggersEnabled(new(spec.VersionControl.EnableFileTriggers))
		attrs.SetTriggerPatterns(spec.VersionControl.TriggerPatterns)
		attrs.SetTriggerPrefixes(spec.VersionControl.TriggerPrefixes)
	}

	if spec.RemoteStateSharing != nil {
		attrs.SetGlobalRemoteState(new(spec.RemoteStateSharing.AllWorkspaces))
	}

	defaultProjectID, err := w.readOrganizationDefaultProjectID(ctx)
	if err != nil {
		w.log.Error(err, "Reconcile Workspace", "msg", "failed to get organization")
		r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to get organization")
		return nil, err
	}

	if spec.Project != nil {
		prjID, err := w.getProjectID(ctx)
		if err != nil {
			w.log.Error(err, "Reconcile Workspace", "msg", "failed to get project ID")
			r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to get project ID")
			return nil, err
		}
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("project ID %s will be used", prjID))
		project := models.NewProjectsHasOne_data()
		project.SetId(new(prjID))
		project.SetTypeEscaped(new(models.PROJECTS_PROJECTSIDENTIFIER_TYPE))
		projectRel := models.NewProjectsHasOne()
		projectRel.SetData(project)
		rels.SetProject(projectRel)
	}

	data := models.NewWorkspaces()
	data.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
	data.SetAttributes(withAgentPoolID(attrs, agentPoolAttr))
	// go-tfe v1 omitted the relationships when none was set.
	if rels.GetProject() != nil {
		data.SetRelationships(rels)
	}
	body := models.NewWorkspacesEnvelope()
	body.SetData(data)

	resp, err := w.tfClient.Client.API.Organizations().ByOrganization_name(spec.Organization).Workspaces().Post(ctx, body, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetData() == nil {
		return nil, fmt.Errorf("workspace creation returned an empty response")
	}
	workspace := resp.GetData()

	patch := client.MergeFrom(w.instance.DeepCopy())
	if defaultProjectID != "" {
		w.instance.Status.DefaultProjectID = defaultProjectID
	}
	w.instance.Status.WorkspaceID = pointer.ValueOf(workspace.GetId())
	if err = r.Status().Patch(ctx, &w.instance, patch); err != nil {
		w.log.Error(err, "Reconcile Workspace", "msg", "failed to update status with workspace ID")
		r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to update status with workspace ID")
		return nil, err
	}
	w.log.Info("Reconcile Workspace", "msg", "successfully updated status with workspace ID")

	return workspace, nil
}

func (r *WorkspaceReconciler) readWorkspace(ctx context.Context, w *workspaceInstance) (models.Workspacesable, error) {
	resp, err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).Get(ctx, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.GetData(), nil
}

// removeWorkspaceVCSConnection detaches the VCS repository from the workspace.
func (w *workspaceInstance) removeWorkspaceVCSConnection(ctx context.Context, workspaceID string) error {
	builder := w.tfClient.Client.API.Workspaces().ByWorkspace_id(workspaceID)
	ri, err := builder.ToPatchRequestInformation(ctx, models.NewWorkspacesEnvelope(), nil)
	if err != nil {
		return err
	}
	// The VCS repository is detached by setting the 'vcs-repo' attribute to null.
	_, err = sendRawJSON(ctx, builder.RequestAdapter, ri, map[string]any{
		"data": map[string]any{
			"type": "workspaces",
			"attributes": map[string]any{
				"vcs-repo": nil,
			},
		},
	})
	return err
}

func (r *WorkspaceReconciler) updateWorkspace(ctx context.Context, w *workspaceInstance, workspace models.Workspacesable) (models.Workspacesable, error) {
	attrs := models.NewWorkspaces_attributes()
	attrs.SetGlobalRemoteState(new(false))
	rels := models.NewWorkspaces_relationships()
	var agentPoolAttr *string
	wsa := workspace.GetAttributes()

	spec := w.instance.Spec
	status := w.instance.Status

	if spec.ExecutionMode == "agent" {
		agentPoolID, err := r.getAgentPoolID(ctx, w)
		if err != nil {
			w.log.Error(err, "Reconcile Workspace", "msg", "failed to get agent pool ID")
			r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to get agent pool ID")
			return nil, err
		}
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("agent pool ID %s will be used", agentPoolID))
		agentPoolAttr = new(agentPoolID)
	}

	if pointer.ValueOf(wsa.GetName()) != spec.Name {
		attrs.SetName(new(spec.Name))
	}

	if pointer.ValueOf(wsa.GetAutoApply()) != ApplyMethodToBool(spec.ApplyMethod) {
		attrs.SetAutoApply(new(ApplyMethodToBool(spec.ApplyMethod)))
	}

	if pointer.ValueOf(wsa.GetAutoApplyRunTrigger()) != ApplyRunTriggerToBool(spec.ApplyRunTrigger) {
		attrs.SetAutoApplyRunTrigger(new(ApplyRunTriggerToBool(spec.ApplyRunTrigger)))
	}

	if pointer.ValueOf(wsa.GetAllowDestroyPlan()) != spec.AllowDestroyPlan {
		attrs.SetAllowDestroyPlan(new(spec.AllowDestroyPlan))
	}

	if pointer.ValueOf(wsa.GetDescription()) != spec.Description {
		attrs.SetDescription(new(spec.Description))
	}

	if wsa.GetExecutionMode() == nil || wsa.GetExecutionMode().String() != spec.ExecutionMode {
		if mode, err := models.ParseWorkspaces_attributes_executionMode(spec.ExecutionMode); err == nil && mode != nil {
			attrs.SetExecutionMode(mode.(*models.Workspaces_attributes_executionMode))
		}
	}

	if spec.RemoteStateSharing != nil {
		attrs.SetGlobalRemoteState(new(spec.RemoteStateSharing.AllWorkspaces))
	}

	if spec.TerraformVersion == "" {
		if pointer.ValueOf(wsa.GetTerraformVersion()) != status.TerraformVersion {
			attrs.SetTerraformVersion(new(status.TerraformVersion))
		}
	} else {
		if pointer.ValueOf(wsa.GetTerraformVersion()) != spec.TerraformVersion {
			attrs.SetTerraformVersion(new(spec.TerraformVersion))
		}
	}

	if pointer.ValueOf(wsa.GetWorkingDirectory()) != spec.WorkingDirectory {
		attrs.SetWorkingDirectory(new(spec.WorkingDirectory))
		attrs.SetName(new(spec.Name))
	}

	if spec.VersionControl == nil && wsa.GetVcsRepo() != nil {
		if err := w.removeWorkspaceVCSConnection(ctx, pointer.ValueOf(workspace.GetId())); err != nil {
			return workspace, err
		}
	}

	if spec.VersionControl != nil {
		vcs := models.NewWorkspaces_attributes_vcsRepo()
		vcs.SetOauthTokenId(new(spec.VersionControl.OAuthTokenID))
		vcs.SetIdentifier(new(spec.VersionControl.Repository))
		vcs.SetBranch(new(spec.VersionControl.Branch))
		attrs.SetVcsRepo(vcs)

		if pointer.ValueOf(wsa.GetSpeculativeEnabled()) != spec.VersionControl.SpeculativePlans {
			attrs.SetSpeculativeEnabled(new(spec.VersionControl.SpeculativePlans))
		}

		if pointer.ValueOf(wsa.GetFileTriggersEnabled()) != spec.VersionControl.EnableFileTriggers {
			attrs.SetFileTriggersEnabled(new(spec.VersionControl.EnableFileTriggers))
		}

		triggerPatternsDiff := vcsTriggersDifference(getWorkspaceTriggerPatterns(workspace), getTriggerPatterns(&w.instance))
		if len(triggerPatternsDiff) != 0 {
			attrs.SetTriggerPatterns(spec.VersionControl.TriggerPatterns)
		}

		triggerPrefixesDiff := vcsTriggersDifference(getWorkspaceTriggerPrefixes(workspace), getTriggerPrefixes(&w.instance))
		if len(triggerPrefixesDiff) != 0 {
			attrs.SetTriggerPrefixes(spec.VersionControl.TriggerPrefixes)
		}
	}

	var projectID string
	if spec.Project != nil {
		prjID, err := w.getProjectID(ctx)
		if err != nil {
			w.log.Error(err, "Reconcile Workspace", "msg", "failed to get project ID")
			r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to get project ID")
			return nil, err
		}
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("project ID %s will be used", prjID))
		projectID = prjID
	} else {
		if w.instance.Status.DefaultProjectID != "" {
			projectID = w.instance.Status.DefaultProjectID
			w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("default project ID %s will be used", w.instance.Status.DefaultProjectID))
		} else {
			defaultProjectID, err := w.readOrganizationDefaultProjectID(ctx)
			if err != nil {
				w.log.Error(err, "Reconcile Workspace", "msg", "failed to get organization")
				r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to get organization")
				return nil, err
			}
			if defaultProjectID != "" {
				w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("default project ID %s will be used", defaultProjectID))
				projectID = defaultProjectID
			}
		}
	}

	if projectID != "" {
		project := models.NewProjectsHasOne_data()
		project.SetId(new(projectID))
		project.SetTypeEscaped(new(models.PROJECTS_PROJECTSIDENTIFIER_TYPE))
		projectRel := models.NewProjectsHasOne()
		projectRel.SetData(project)
		rels.SetProject(projectRel)
	}

	data := models.NewWorkspaces()
	data.SetTypeEscaped(new(models.WORKSPACES_WORKSPACES_TYPE))
	data.SetAttributes(withAgentPoolID(attrs, agentPoolAttr))
	// go-tfe v1 omitted the relationships when none was set.
	if rels.GetProject() != nil {
		data.SetRelationships(rels)
	}
	body := models.NewWorkspacesEnvelope()
	body.SetData(data)

	resp, err := w.tfClient.Client.API.Workspaces().ByWorkspace_id(w.instance.Status.WorkspaceID).Patch(ctx, body, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	return resp.GetData(), nil
}

func (r *WorkspaceReconciler) reconcileWorkspace(ctx context.Context, w *workspaceInstance) error {
	w.log.Info("Reconcile Workspace", "msg", "reconciling workspace")

	var workspace models.Workspacesable
	var err error

	defer func() {
		// Update the status with the Workspace ID. This is useful if the reconciliation failed.
		// An example here would be the case when the workspace has been created successfully,
		// but further reconciliation steps failed.
		//
		// If a Workspace creation operation failed, we don't have a workspace object
		// and thus don't update the status. An example here would be the case when the workspace name has already been taken.
		//
		// Cannot call updateStatus method since it updated multiple fields and can break reconciliation logic.
		//
		// TODO:
		// - Use conditions(https://maelvls.dev/kubernetes-conditions/)
		// - Let Objects update their own status conditions
		// - Simplify updateStatus method in a way it could be called anytime
		if workspace != nil && pointer.ValueOf(workspace.GetId()) != "" {
			patch := client.MergeFrom(w.instance.DeepCopy())
			w.instance.Status.WorkspaceID = pointer.ValueOf(workspace.GetId())
			if err := r.Status().Patch(ctx, &w.instance, patch); err != nil {
				w.log.Error(err, "Workspace Controller", "msg", "failed to update status with workspace ID")
				r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to update status with workspace ID")
			}
			w.log.Info("Reconcile Workspace", "msg", "successfully updated status with workspace ID")
		}
	}()

	// verify whether the Kubernetes object has been marked as deleted and if so delete the workspace
	if isDeletionCandidate(&w.instance, workspaceFinalizer) {
		w.log.Info("Reconcile Workspace", "msg", "object marked as deleted, need to delete workspace first")
		r.Recorder.Event(&w.instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Object marked as deleted, need to delete workspace first")
		return r.deleteWorkspace(ctx, w)
	}

	// create a new workspace if workspace ID is unknown(means it was never created by the controller)
	// this condition will work just one time, when a new Kubernetes object is created
	if w.instance.IsCreationCandidate() {
		w.log.Info("Reconcile Workspace", "msg", "status.WorkspaceID is empty, creating a new workspace")
		r.Recorder.Event(&w.instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Status.WorkspaceID is empty, creating a new workspace")
		workspace, err = r.createWorkspace(ctx, w)
		if err != nil {
			w.log.Error(err, "Reconcile Workspace", "msg", "failed to create a new workspace")
			r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to create a new workspace")
			return err
		}
		w.log.Info("Reconcile Workspace", "msg", "successfully created a new workspace")
		r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Successfully created a new workspace with ID %s", w.instance.Status.WorkspaceID)
	}

	// read the HCP Terraform workspace to compare it with the Kubernetes object spec
	workspace, err = r.readWorkspace(ctx, w)
	if err != nil {
		// 'ResourceNotFound' means that the TF Cloud workspace was removed from the TF Cloud bypass the operator
		if errors.Is(err, tfc.ErrNotFound) {
			w.log.Info("Reconcile Workspace", "msg", "workspace not found, creating a new workspace")
			r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Workspace ID %s not found, creating a new workspace", w.instance.Status.WorkspaceID)
			workspace, err = r.createWorkspace(ctx, w)
			if err != nil {
				w.log.Error(err, "Reconcile Workspace", "msg", "failed to create a new workspace")
				r.Recorder.Event(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to create a new workspace")
				return err
			}
			w.log.Info("Reconcile Workspace", "msg", "successfully created a new workspace")
			r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Successfully created a new workspace with ID %s", w.instance.Status.WorkspaceID)
		} else {
			w.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to read workspace ID %s", w.instance.Status.WorkspaceID))
			r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to read workspace ID %s", w.instance.Status.WorkspaceID)
			return err
		}
	}

	// update workspace if any changes have been made in the Kubernetes object spec or HCP Terraform workspace
	if needToUpdateWorkspace(&w.instance, workspace) {
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("observed and desired states are not matching, need to update workspace ID %s", w.instance.Status.WorkspaceID))
		workspace, err = r.updateWorkspace(ctx, w, workspace)
		if err != nil {
			w.log.Error(err, "Reconcile Workspace", "msg", fmt.Sprintf("failed to update workspace ID %s", w.instance.Status.WorkspaceID))
			r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileWorkspace", "Failed to update workspace ID %s", w.instance.Status.WorkspaceID)
			return err
		}
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("Successfully updated workspace ID %s", w.instance.Status.WorkspaceID))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileWorkspace", "Successfully updated workspace ID %s", w.instance.Status.WorkspaceID)
	} else {
		w.log.Info("Reconcile Workspace", "msg", fmt.Sprintf("observed and desired states are matching, no need to update workspace ID %s", w.instance.Status.WorkspaceID))
	}

	// reconcile SSH key
	err = w.reconcileSSHKey(ctx, workspace)
	if err != nil {
		w.log.Error(err, "Reconcile SSH Key", "msg", "failed to assign ssh key ID")
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileSSHKey", "Failed to assign SSH Key ID")
		return err
	} else {
		w.log.Info("Reconcile SSH Key", "msg", "successfully reconcile ssh key")
		r.Recorder.Event(&w.instance, corev1.EventTypeNormal, "ReconcileSSHKey", "Successfully reconcile SSH Key")
	}

	// Reconcile Tags
	err = r.reconcileTags(ctx, w, workspace)
	if err != nil {
		w.log.Error(err, "Reconcile Tags", "msg", "reconcile tags")
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileTags", "Failed to reconcile tags in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Tags", "msg", "successfully reconcilied tags")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileTags", "Successfully reconcilied tags in workspace ID %s", w.instance.Status.WorkspaceID)

	// Reconcile Variables
	err = r.reconcileVariables(ctx, w, workspace)
	if err != nil {
		w.log.Error(err, "Reconcile Variables", "msg", fmt.Sprintf("failed to reconcile variables in workspace ID %s", w.instance.Status.WorkspaceID))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileVariables", "Failed to reconcile variables in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Variables", "msg", "successfully reconcilied variables")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileVariables", "Reconcilied variables in workspace ID %s", w.instance.Status.WorkspaceID)

	// Reconcile Variable Sets
	err = r.reconcileVariableSets(ctx, w)
	if err != nil {
		w.log.Info("Reconcile Variable Sets", "msg", fmt.Sprintf("failed to reconcile variable sets in workspace ID %s", w.instance.Status.WorkspaceID))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileVariableSets", "Failed to reconcile variable sets in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Variable Sets", "msg", "successfully reconciled variable sets")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileVariableSets", "Successfully reconciled variable sets in workspace ID %s", w.instance.Status.WorkspaceID)

	// Reconcile Run Triggers
	err = r.reconcileRunTriggers(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Run Triggers", "msg", fmt.Sprintf("failed to reconcile run triggers in workspace ID %s", w.instance.Status.WorkspaceID))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileRunTriggers", "Failed to reconcile run triggers in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Run Triggers", "msg", "successfully reconcilied run triggers")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileRunTriggers", "Reconcilied run triggers in workspace ID %s", w.instance.Status.WorkspaceID)

	// Reconcile Team Access
	err = r.reconcileTeamAccess(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Team Access", "msg", fmt.Sprintf("failed to reconcile team access in workspace ID %s", w.instance.Status.WorkspaceID))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileTeamAccess", "Failed to reconcile team access in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Team Access", "msg", "successfully reconcilied team access")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileTeamAccess", "Reconcilied team access in workspace ID %s", w.instance.Status.WorkspaceID)

	// Reconcile Remote State Sharing
	err = r.reconcileRemoteStateSharing(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Remote State Sharing", "msg", fmt.Sprintf("failed to reconcile remote state sharing in workspace ID %s", w.instance.Status.WorkspaceID))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileRemoteStateSharing", "Failed to reconcile remote state sharing in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Remote State Sharing", "msg", "successfully reconcilied remote state sharing")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileRemoteStateSharing", "Reconcilied remote state sharing in workspace ID %s", w.instance.Status.WorkspaceID)

	// Reconcile Run Tasks
	err = r.reconcileRunTasks(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Run Tasks", "msg", fmt.Sprintf("failed to reconcile run tasks in workspace ID %s", w.instance.Status.WorkspaceID))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileRunTasks", "Failed to reconcile run tasks in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Run Tasks", "msg", "successfully reconcilied run tasks")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileRunTasks", "Reconcilied run tasks in workspace ID %s", w.instance.Status.WorkspaceID)

	// Reconcile Notifications
	err = r.reconcileNotifications(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Notifications", "msg", fmt.Sprintf("failed to reconcile notifications in workspace ID %s", w.instance.Status.WorkspaceID))
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileNotifications", "Failed to reconcile notifications in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Notifications", "msg", "successfully reconcilied notifications")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileNotifications", "Reconcilied notifications in workspace ID %s", w.instance.Status.WorkspaceID)

	// Reconcile Runs (Status)
	// This reconciliation should always happen before `reconcileOutputs`
	err = r.reconcileRuns(ctx, w, workspace)
	if err != nil {
		w.log.Error(err, "Reconcile Runs", "msg", "failed to reconcile runs")
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileRuns", "Failed to reconcile runs in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Runs", "msg", "successfully reconcilied runs")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileRuns", "Successfully reconcilied runs in workspace ID %s", w.instance.Status.WorkspaceID)

	// Reconcile Outputs
	// This reconciliation should always happen after `reconcileRuns`
	// TODO:
	// - move the output reconciliation to runs reconciliation since they depend on each other
	err = r.reconcileOutputs(ctx, w)
	if err != nil {
		w.log.Error(err, "Reconcile Outputs", "msg", "failed to reconcile outputs")
		r.Recorder.Eventf(&w.instance, corev1.EventTypeWarning, "ReconcileOutputs", "Failed to reconcile outputs in workspace ID %s", w.instance.Status.WorkspaceID)
		return err
	}
	w.log.Info("Reconcile Outputs", "msg", "successfully reconcilied outputs")
	r.Recorder.Eventf(&w.instance, corev1.EventTypeNormal, "ReconcileOutputs", "Successfully reconcilied outputs in workspace ID %s", w.instance.Status.WorkspaceID)

	return r.updateStatus(ctx, w, workspace)
}
