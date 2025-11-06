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
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/version"
)

var runStatuses = []tfc.RunStatus{
	tfc.RunApplied,
	tfc.RunApplying,
	tfc.RunApplyQueued,
	tfc.RunCanceled,
	tfc.RunConfirmed,
	tfc.RunCostEstimated,
	tfc.RunCostEstimating,
	tfc.RunDiscarded,
	tfc.RunErrored,
	tfc.RunFetching,
	tfc.RunFetchingCompleted,
	tfc.RunPending,
	tfc.RunPlanned,
	tfc.RunPlannedAndFinished,
	tfc.RunPlannedAndSaved,
	tfc.RunPlanning,
	tfc.RunPlanQueued,
	tfc.RunPolicyChecked,
	tfc.RunPolicyChecking,
	tfc.RunPolicyOverride,
	tfc.RunPolicySoftFailed,
	tfc.RunPostPlanAwaitingDecision,
	tfc.RunPostPlanCompleted,
	tfc.RunPostPlanRunning,
	tfc.RunPreApplyRunning,
	tfc.RunPreApplyCompleted,
	tfc.RunPrePlanCompleted,
	tfc.RunPrePlanRunning,
	tfc.RunQueuing,
	tfc.RunQueuingApply,
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
		if errors.IsNotFound(err) {
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

	if a, ok := rc.instance.GetAnnotations()[annotationPaused]; ok && a == metaTrue {
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

	httpClient := tfc.DefaultConfig().HTTPClient
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
func (r *RunsCollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1alpha2.RunsCollector{}).
		WithEventFilter(predicate.Or(genericPredicates())).
		Complete(r)
}

func (r *RunsCollectorReconciler) getAgentPoolIDByName(ctx context.Context, rc *runsCollectorInstance) (*tfc.AgentPool, error) {
	name := rc.instance.Spec.AgentPool.Name

	listOpts := &tfc.AgentPoolListOptions{
		Query: name,
		ListOptions: tfc.ListOptions{
			PageSize: maxPageSize,
		},
	}
	for {
		ap, err := rc.tfClient.Client.AgentPools.List(ctx, rc.instance.Spec.Organization, listOpts)
		if err != nil {
			return nil, err
		}
		for _, a := range ap.Items {
			if a.Name == name {
				return a, nil
			}
		}
		if ap.NextPage == 0 {
			break
		}
		listOpts.PageNumber = ap.NextPage
	}

	return nil, fmt.Errorf("agent pool ID not found for agent pool name %q", name)
}

func (r *RunsCollectorReconciler) updateStatusAgentPool(ctx context.Context, rc *runsCollectorInstance) error {
	var pool *tfc.AgentPool
	var err error
	if rc.instance.Spec.AgentPool.Name != "" {
		pool, err = r.getAgentPoolIDByName(ctx, rc)
		if err != nil {
			return err
		}

	}
	if rc.instance.Spec.AgentPool.ID != "" {
		pool, err = rc.tfClient.Client.AgentPools.Read(ctx, rc.instance.Spec.AgentPool.ID)
		if err != nil {
			return err
		}

	}
	rc.instance.Status.AgentPool = &appv1alpha2.AgentPoolRef{
		ID:   pool.ID,
		Name: pool.Name,
	}

	return nil
}

func (r *RunsCollectorReconciler) reconcileRuns(ctx context.Context, rc *runsCollectorInstance) error {
	runs := map[tfc.RunStatus]float64{}
	var runsTotal float64
	if rc.instance.NeedUpdateStatus() {
		r.updateStatusAgentPool(ctx, rc)
	}

	listOpts := &tfc.RunListForOrganizationOptions{
		AgentPoolNames: rc.instance.Status.AgentPool.Name,
		StatusGroup:    "non_final",
		ListOptions: tfc.ListOptions{
			PageSize:   maxPageSize,
			PageNumber: initPageNumber,
		},
	}

	for {
		runsList, err := rc.tfClient.Client.Runs.ListForOrganization(ctx, rc.instance.Spec.Organization, listOpts)
		// TODO:
		// - Think if we need to reset all metrics to 0 in case of error.
		if err != nil {
			return err
		}
		runsTotal += float64(len(runsList.Items))
		for _, run := range runsList.Items {
			runs[run.Status]++
		}
		if runsList.NextPage == 0 {
			break
		}
		listOpts.PageNumber = runsList.NextPage
	}

	for _, status := range runStatuses {
		metricRuns.WithLabelValues(string(status)).Set(float64(runs[status]))
	}

	rc.log.Info("Reconcile Runs Collector", "msg", fmt.Sprintf("Total Runs: %.0f", runsTotal))
	metricRunsTotal.WithLabelValues().Set(runsTotal)

	rc.instance.Status.ObservedGeneration = rc.instance.Generation

	return r.Status().Update(ctx, &rc.instance)
}

// TODO:
// - Since we have one CR per Agent Pool, we can implement an aggregation logic
//   to to perform one API call to collect runs for all Agent Pools instead of one
//   call per CR.
