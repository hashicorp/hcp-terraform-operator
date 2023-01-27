// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/go-logr/zapr"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
	"github.com/hashicorp/terraform-cloud-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(appv1alpha2.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	// GLOBAL OPTIONS
	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	var syncPeriod time.Duration
	flag.DurationVar(&syncPeriod, "sync-period", 5*time.Minute,
		"The minimum frequency at which watched resources are reconciled. Format: 5s, 1m, etc.")
	var watchNamespaces cliNamespaces
	flag.Var(&watchNamespaces, "namespace", "Namespace to watch")
	// OPERATOR OPRTIONS
	var agentPoolWorkers int
	flag.IntVar(&agentPoolWorkers, "agent-pool-workers", 1,
		"The number of the Agent Pool controller workers.")
	var workspaceWorkers int
	flag.IntVar(&workspaceWorkers, "workspace-workers", 1,
		"The number of the Workspace controller workers.")
	var moduleWorkers int
	flag.IntVar(&moduleWorkers, "module-workers", 1,
		"The number of the Module controller workers.")

	flag.Parse()

	zapConfig := zap.NewProductionConfig()
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	zapConfig.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	zapConfig.Encoding = "console"
	zapConfig.DisableCaller = true
	zapConfig.DisableStacktrace = true

	logger, err := zapConfig.Build(zap.AddStacktrace(zapcore.DPanicLevel))
	if err != nil {
		setupLog.Error(err, "unable to set up logging")
		os.Exit(1)
	}
	ctrl.SetLogger(zapr.NewLogger(logger))

	options := ctrl.Options{
		Controller: v1alpha1.ControllerConfigurationSpec{
			GroupKindConcurrency: map[string]int{
				"Workspace.app.terraform.io": workspaceWorkers,
				"Module.app.terraform.io":    moduleWorkers,
				"AgentPool.app.terraform.io": agentPoolWorkers,
			},
		},
		Scheme:     scheme,
		SyncPeriod: &syncPeriod,
	}
	if configFile != "" {
		options, err = options.AndFrom(ctrl.ConfigFile().AtPath(configFile))
		if err != nil {
			setupLog.Error(err, "unable to load the config file")
			os.Exit(1)
		}
	}
	if len(watchNamespaces) != 0 {
		setupLog.Info("Watching namespaces: " + strings.Join(watchNamespaces, " "))
		options.NewCache = cache.MultiNamespacedCacheBuilder(watchNamespaces)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.WorkspaceReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("WorkspaceController"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Workspace")
		os.Exit(1)
	}
	if err = (&controllers.ModuleReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("ModuleController"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Module")
		os.Exit(1)
	}
	if err = (&controllers.AgentPoolReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("AgentPoolController"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AgentPool")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

type cliNamespaces []string

func (n *cliNamespaces) String() string {
	return strings.Join(*n, ",")
}

func (n *cliNamespaces) Set(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("namespace cannot be empty")
	}
	for _, v := range *n {
		if v == s {
			return nil
		}
	}
	*n = append(*n, s)
	return nil
}
