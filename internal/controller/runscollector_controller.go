// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/go-logr/logr"
	tfc "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/hashicorp/go-tfe/v2/api/organizations"
	abstractions "github.com/microsoft/kiota-abstractions-go"
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

var runStatuses = []string{
	models.APPLIED_RUNS_ATTRIBUTES_STATUS.String(),
	models.APPLYING_RUNS_ATTRIBUTES_STATUS.String(),
	models.APPLY_QUEUED_RUNS_ATTRIBUTES_STATUS.String(),
	models.CANCELED_RUNS_ATTRIBUTES_STATUS.String(),
	models.CONFIRMED_RUNS_ATTRIBUTES_STATUS.String(),
	models.COST_ESTIMATED_RUNS_ATTRIBUTES_STATUS.String(),
	models.COST_ESTIMATING_RUNS_ATTRIBUTES_STATUS.String(),
	models.DISCARDED_RUNS_ATTRIBUTES_STATUS.String(),
	models.ERRORED_RUNS_ATTRIBUTES_STATUS.String(),
	models.FETCHING_RUNS_ATTRIBUTES_STATUS.String(),
	models.FETCHING_COMPLETED_RUNS_ATTRIBUTES_STATUS.String(),
	models.PENDING_RUNS_ATTRIBUTES_STATUS.String(),
	models.PLANNED_RUNS_ATTRIBUTES_STATUS.String(),
	models.PLANNED_AND_FINISHED_RUNS_ATTRIBUTES_STATUS.String(),
	models.PLANNED_AND_SAVED_RUNS_ATTRIBUTES_STATUS.String(),
	models.PLANNING_RUNS_ATTRIBUTES_STATUS.String(),
	models.PLAN_QUEUED_RUNS_ATTRIBUTES_STATUS.String(),
	models.POLICY_CHECKED_RUNS_ATTRIBUTES_STATUS.String(),
	models.POLICY_CHECKING_RUNS_ATTRIBUTES_STATUS.String(),
	models.POLICY_OVERRIDE_RUNS_ATTRIBUTES_STATUS.String(),
	models.POLICY_SOFT_FAILED_RUNS_ATTRIBUTES_STATUS.String(),
	appv1alpha2.RunPostPlanAwaitingDecision,
	models.POST_PLAN_COMPLETED_RUNS_ATTRIBUTES_STATUS.String(),
	models.POST_PLAN_RUNNING_RUNS_ATTRIBUTES_STATUS.String(),
	models.PRE_APPLY_RUNNING_RUNS_ATTRIBUTES_STATUS.String(),
	models.PRE_APPLY_COMPLETED_RUNS_ATTRIBUTES_STATUS.String(),
	models.PRE_PLAN_COMPLETED_RUNS_ATTRIBUTES_STATUS.String(),
	models.PRE_PLAN_RUNNING_RUNS_ATTRIBUTES_STATUS.String(),
	models.QUEUING_RUNS_ATTRIBUTES_STATUS.String(),
	models.QUEUING_APPLY_RUNS_ATTRIBUTES_STATUS.String(),
}

// RunsCollectorReconciler reconciles a RunsCollector object
type RunsCollectorReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Scheme   *runtime.Scheme
}

type runsCollectorInstance struct {
	instance appv1alpha2.RunsCollector

	log      logr.Logger
	tfClient HCPTerraformClient
}

//+kubebuilder:rbac:groups=app.terraform.io,resources=runscollectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.terraform.io,resources=runscollectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.terraform.io,resources=runscollectors/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *RunsCollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	rc := runsCollectorInstance{}

	rc.log = log.Log.WithValues("runscollector", req.NamespacedName)
	rc.log.Info("Runs Collector Controller", "msg", "new reconciliation event")

	err := r.Client.Get(ctx, req.NamespacedName, &rc.instance)
	if err != nil {
		// 'Not found' error occurs when an object is removed from the Kubernetes
		// No actions are required in this case
		if kerrors.IsNotFound(err) {
			rc.log.Info("Runs Collector Controller", "msg", "the instance was removed no further action is required")
			return doNotRequeue()
		}
		rc.log.Error(err, "Runs Collector Controller", "msg", "get instance object")
		return requeueAfter(requeueInterval)
	}

	// TODO:
	// - Think about using the DeleteFunc predicate.
	if rc.instance.DeletionTimestamp != nil && !controllerutil.ContainsFinalizer(&rc.instance, runsCollectorFinalizer) {
		rc.log.Info("Runs Collector Controller", "msg", "object marked as deleted without finalizer, no further action is required")
		return doNotRequeue()
	}

	if a, ok := rc.instance.GetAnnotations()[annotationPaused]; ok && a == MetaTrue {
		rc.log.Info("Runs Collector Controller", "msg", "reconciliation is paused for this resource")
		return doNotRequeue()
	}

	rc.log.Info("Spec Validation", "msg", "validating instance object spec")
	if err := rc.instance.ValidateSpec(); err != nil {
		rc.log.Error(err, "Spec Validation", "msg", "spec is invalid, exit from reconciliation")
		r.Recorder.Event(&rc.instance, corev1.EventTypeWarning, "SpecValidation", err.Error())
		return doNotRequeue()
	}
	rc.log.Info("Spec Validation", "msg", "spec is valid")

	if needToAddFinalizer(&rc.instance, runsCollectorFinalizer) {
		err := r.addFinalizer(ctx, &rc.instance)
		if err != nil {
			rc.log.Error(err, "Runs Collector Controller", "msg", fmt.Sprintf("failed to add finalizer %s to the object", runsCollectorFinalizer))
			r.Recorder.Eventf(&rc.instance, corev1.EventTypeWarning, "AddFinalizer", "Failed to add finalizer %s to the object", runsCollectorFinalizer)
			return requeueOnErr(err)
		}
		rc.log.Info("Runs Collector Controller", "msg", fmt.Sprintf("successfully added finalizer %s to the object", runsCollectorFinalizer))
		r.Recorder.Eventf(&rc.instance, corev1.EventTypeNormal, "AddFinalizer", "Successfully added finalizer %s to the object", runsCollectorFinalizer)
	}

	err = r.getTerraformClient(ctx, &rc)
	if err != nil {
		rc.log.Error(err, "Runs Collector Controller", "msg", "failed to get HCP Terraform client")
		r.Recorder.Event(&rc.instance, corev1.EventTypeWarning, "TerraformClient", "Failed to get HCP Terraform Client")
		return requeueAfter(requeueInterval)
	}

	err = r.reconcileRuns(ctx, &rc)
	if err != nil {
		rc.log.Error(err, "Runs Collector Controller", "msg", "Reconcile Runs")
		r.Recorder.Event(&rc.instance, corev1.EventTypeWarning, "ReconcileRunsCollector", "Failed to Reconcile Runs")
		return requeueAfter(requeueInterval)
	}
	rc.log.Info("Runs Collector Controller", "msg", "successfully reconcilied runs")
	r.Recorder.Event(&rc.instance, corev1.EventTypeNormal, "ReconcileRunsCollector", "Successfully reconcilied runs")

	return requeueAfter(RunsCollectorSyncPeriod)
}

func (r *RunsCollectorReconciler) addFinalizer(ctx context.Context, instance *appv1alpha2.RunsCollector) error {
	controllerutil.AddFinalizer(instance, runsCollectorFinalizer)

	return r.Update(ctx, instance)
}

func (r *RunsCollectorReconciler) getTerraformClient(ctx context.Context, t *runsCollectorInstance) error {
	nn := types.NamespacedName{
		Namespace: t.instance.Namespace,
		Name:      t.instance.Spec.Token.SecretKeyRef.Name,
	}
	token, err := secretKeyRef(ctx, r.Client, nn, t.instance.Spec.Token.SecretKeyRef.Key)
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
		t.log.Info("Reconcile Runs Collector", "msg", "client configured to skip TLS certificate verifications")
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
	t.tfClient.Client, err = tfc.NewClient(config)

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *RunsCollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha2.RunsCollector{}).
		WithEventFilter(predicate.Or(genericPredicates())).
		Complete(r)
}

func (r *RunsCollectorReconciler) removeFinalizer(ctx context.Context, rc *runsCollectorInstance) error {
	controllerutil.RemoveFinalizer(&rc.instance, runsCollectorFinalizer)

	err := r.Update(ctx, &rc.instance)
	if err != nil {
		rc.log.Error(err, "Reconcile Runs Collector", "msg", fmt.Sprintf("failed to remove finalizer %s", runsCollectorFinalizer))
		r.Recorder.Eventf(&rc.instance, corev1.EventTypeWarning, "RemoveRunsCollector", "Failed to remove finalizer %s", runsCollectorFinalizer)
	}

	return err
}

func (r *RunsCollectorReconciler) getAgentPoolIDByName(ctx context.Context, rc *runsCollectorInstance) (*appv1alpha2.AgentPoolRef, error) {
	name := rc.instance.Spec.AgentPool.Name

	qp := &organizations.ItemAgentPoolsRequestBuilderGetQueryParameters{
		Q:        &name,
		Pagesize: new(int32(MaxPageSize)),
	}
	pageNumber := int32(InitPageNumber)
	for {
		qp.Pagenumber = &pageNumber
		ap, err := rc.tfClient.Client.API.Organizations().ByOrganization_name(rc.instance.Spec.Organization).AgentPools().Get(ctx, &abstractions.RequestConfiguration[organizations.ItemAgentPoolsRequestBuilderGetQueryParameters]{
			QueryParameters: qp,
		})
		if err != nil {
			return nil, err
		}
		for _, a := range ap.GetData() {
			if a.GetAttributes() != nil && pointer.ValueOf(a.GetAttributes().GetName()) == name {
				return &appv1alpha2.AgentPoolRef{ID: pointer.ValueOf(a.GetId()), Name: name}, nil
			}
		}
		next := nextPage(ap.GetMeta())
		if next == 0 {
			break
		}
		pageNumber = next
	}

	return nil, fmt.Errorf("agent pool ID not found for agent pool name %q", name)
}

func (r *RunsCollectorReconciler) updateStatusAgentPool(ctx context.Context, rc *runsCollectorInstance) error {
	var pool *appv1alpha2.AgentPoolRef
	var err error
	if rc.instance.Spec.AgentPool.Name != "" {
		pool, err = r.getAgentPoolIDByName(ctx, rc)
		if err != nil {
			return err
		}

	}
	if rc.instance.Spec.AgentPool.ID != "" {
		resp, err := rc.tfClient.Client.API.AgentPools().ByAgent_pool_id(rc.instance.Spec.AgentPool.ID).Get(ctx, nil)
		if err != nil {
			return err
		}
		pool = &appv1alpha2.AgentPoolRef{ID: pointer.ValueOf(resp.GetData().GetId())}
		if resp.GetData().GetAttributes() != nil {
			pool.Name = pointer.ValueOf(resp.GetData().GetAttributes().GetName())
		}
	}
	rc.instance.Status.AgentPool = pool

	return nil
}

func (r *RunsCollectorReconciler) reconcileRuns(ctx context.Context, rc *runsCollectorInstance) error {
	// TODO:
	// - Think if we need to reset all metrics to 0 when remove finalizer.
	if isDeletionCandidate(&rc.instance, runsCollectorFinalizer) {
		rc.log.Info("Reconcile Runs Collector", "msg", "object marked as deleted, need to remove finalizer")
		r.Recorder.Event(&rc.instance, corev1.EventTypeNormal, "ReconcileRunsCollector", "Object marked as deleted, need to remove finalizer")
		return r.removeFinalizer(ctx, rc)
	}

	runs := map[string]float64{}
	var runsTotal float64
	// TODO:
	// - If a new name or ID is set in the spec, we need to zero related metrics.
	if rc.instance.NeedUpdateStatus() {
		r.updateStatusAgentPool(ctx, rc)
	}

	pageNumber := int32(InitPageNumber)
	for {
		// The generated API client does not expose the 'filter[agent_pool_names]' and 'filter[status_group]'
		// query parameters, so a raw URL is used.
		u := rc.tfClient.Client.BaseURL()
		runsURL := u.JoinPath("organizations", rc.instance.Spec.Organization, "runs")
		q := url.Values{}
		if rc.instance.Status.AgentPool.Name != "" {
			q.Set("filter[agent_pool_names]", rc.instance.Status.AgentPool.Name)
		}
		q.Set("filter[status_group]", "non_final")
		q.Set("page[size]", fmt.Sprint(MaxPageSize))
		q.Set("page[number]", fmt.Sprint(pageNumber))
		runsURL.RawQuery = q.Encode()
		runsList, err := rc.tfClient.Client.API.Organizations().
			ByOrganization_name(rc.instance.Spec.Organization).
			Runs().
			WithUrl(runsURL.String()).
			Get(ctx, nil)
		// TODO:
		// - Think if we need to reset all metrics to 0 in case of error.
		if err != nil {
			return err
		}
		runsTotal += float64(len(runsList.GetData()))
		for _, run := range runsList.GetData() {
			runs[runAttributesStatus(run.GetAttributes())]++
		}
		next := nextPage(runsList.GetMeta())
		if next == 0 {
			break
		}
		pageNumber = next
	}

	for _, status := range runStatuses {
		MetricRuns.WithLabelValues(
			string(status),                    // run_status
			rc.instance.Status.AgentPool.ID,   // agent_pool_id
			rc.instance.Status.AgentPool.Name, // agent_pool_name
		).Set(float64(runs[status]))
	}

	rc.log.Info("Reconcile Runs Collector", "msg", fmt.Sprintf("Total Runs: %.0f", runsTotal))
	MetricRunsTotal.WithLabelValues(
		rc.instance.Status.AgentPool.ID,   // agent_pool_id
		rc.instance.Status.AgentPool.Name, // agent_pool_name
	).Set(runsTotal)

	rc.instance.Status.ObservedGeneration = rc.instance.Generation

	return r.Status().Update(ctx, &rc.instance)
}

// TODO:
// - Since we have one CR per Agent Pool, we can implement an aggregation logic
//   to to perform one API call to collect runs for all Agent Pools instead of one
//   call per CR.
