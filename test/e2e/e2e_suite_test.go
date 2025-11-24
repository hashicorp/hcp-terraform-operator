// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/controller"
	"github.com/hashicorp/hcp-terraform-operator/version"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cancel context.CancelFunc
var ctx context.Context
var cfg *rest.Config
var k8sClient client.Client
var testEnv = envtest.Environment{}
var tfClient *tfc.Client

var organization = os.Getenv("TFC_ORG")
var terraformToken = os.Getenv("TFC_TOKEN")
var tfcDefaultAddress = "app.terraform.io"
var cloudEndpoint = tfcDefaultAddress

var metricsBindAddress = "127.0.0.1:8080"

var syncPeriod = 30 * time.Second

var secretKey = "token"
var dummySecretKey = "dummy"
var secretNamespacedName = types.NamespacedName{
	Name:      "this",
	Namespace: metav1.NamespaceDefault,
}

var rndm = rand.New(rand.NewSource(GinkgoRandomSeed()))

// TODO:
//   - Find a different way to create an ephemeral webhook URL as a more reliable solution.
//     Perhaps, use smee client to create a new channel per test run.
var webhookURL = "https://smee.io/PZeBPogfii2vPl6s"

func TestControllersAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	suiteConfig, reporterConfig := GinkgoConfiguration()

	reporterConfig.NoColor = true
	reporterConfig.Succinct = false

	RunSpecs(t, "Controllers Suite", suiteConfig, reporterConfig)
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("Set up endpoint")
	if v, ok := os.LookupEnv("TFE_ADDRESS"); ok {
		u, err := url.Parse(v)
		if err != nil {
			Fail("Cannot get hostname from the URL provided in TFE_ADDRESS")
		}
		cloudEndpoint = u.Host
	}

	By("bootstrapping test environment")
	if os.Getenv("USE_EXISTING_CLUSTER") == "true" {
		b := true
		testEnv.UseExistingCluster = &b
	} else {
		testEnv.CRDDirectoryPaths = []string{filepath.Join("..", "..", "config", "crd", "bases")}
		testEnv.ErrorIfCRDPathMissing = true
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = appv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())

	controller.RegisterMetrics()

	//+kubebuilder:scaffold:scheme

	if organization == "" {
		Fail("Environment variable TFC_ORG is required, but either not set or empty")
	}
	if terraformToken == "" {
		Fail("Environment variable TFC_TOKEN is required, but either not set or empty")
	}
	// HCP Terraform Client
	httpClient := tfc.DefaultConfig().HTTPClient
	insecure := false
	if v, ok := os.LookupEnv("TFC_TLS_SKIP_VERIFY"); ok {
		insecure, err = strconv.ParseBool(v)
		if err != nil {
			Fail(fmt.Sprintf("Cannot convert value of TFC_TLS_SKIP_VERIFY into a bool: %v", err))
		}
	}
	fmt.Fprintf(GinkgoWriter, "TFC_TLS_SKIP_VERIFY: %v", insecure)
	httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	tfcConfig := &tfc.Config{
		Token:      os.Getenv("TFC_TOKEN"),
		HTTPClient: httpClient,
		Headers: http.Header{
			"User-Agent": []string{version.UserAgent},
		},
	}
	tfClient, err = tfc.NewClient(tfcConfig)
	Expect(err).Should(Succeed())
	Expect(tfClient).ToNot(BeNil())

	// Kubernetes Client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	if os.Getenv("USE_EXISTING_CLUSTER") != "true" {
		By("starting Kubernetes manager")
		// Kubernetes Manager
		k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme.Scheme,
			Cache: cache.Options{
				SyncPeriod: &syncPeriod,
			},
			Controller: config.Controller{
				GroupKindConcurrency: map[string]int{
					"AgentPool.app.terraform.io":     5,
					"AgentToken.app.terraform.io":    5,
					"Module.app.terraform.io":        5,
					"Project.app.terraform.io":       5,
					"RunsCollector.app.terraform.io": 5,
					"Workspace.app.terraform.io":     5,
				},
			},
			Metrics: server.Options{
				BindAddress: metricsBindAddress,
			},
		})
		Expect(err).ToNot(HaveOccurred())

		err = (&controller.AgentPoolReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("AgentPoolController"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		err = (&controller.AgentTokenReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("AgentTokenController"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		err = (&controller.ModuleReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("ModuleController"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		err = (&controller.ProjectReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("ProjectController"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		err = (&controller.RunsCollectorReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("RunsCollectorController"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		err = (&controller.WorkspaceReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("WorkspaceController"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			err = k8sManager.Start(ctx)
			Expect(err).ToNot(HaveOccurred(), "failed to run manager")
		}()
	}

	// Create a secret object with a TFC token that will be used by the controller
	err = k8sClient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNamespacedName.Name,
			Namespace: secretNamespacedName.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretKey:      []byte(terraformToken),
			dummySecretKey: []byte(dummySecretKey),
		},
	})
	Expect(err).ToNot(HaveOccurred(), "failed to create a token secret")
})

var _ = AfterSuite(func() {
	// DELETE SECRET ONCE ALL TESTS ARE DONE
	// WORKS WHEN RUN ON EXISTING CLUSTER
	err := k8sClient.Delete(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretNamespacedName.Name,
			Namespace: secretNamespacedName.Namespace,
		},
	})
	Expect(err).ToNot(HaveOccurred(), "failed to delete a token secret")

	cancel()
	By("tearing down the test environment")
	err = testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

type TestObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func (in *TestObject) DeepCopyObject() runtime.Object {
	return nil
}

func randomNumber() int32 {
	GinkgoHelper()
	return rndm.Int31()
}

func newNamespacedName() types.NamespacedName {
	GinkgoHelper()
	return types.NamespacedName{
		Name:      fmt.Sprintf("this-%v", randomNumber()),
		Namespace: metav1.NamespaceDefault,
	}
}

func getNamespacedName[T controller.Object](o T) types.NamespacedName {
	GinkgoHelper()
	return types.NamespacedName{
		Namespace: o.GetNamespace(),
		Name:      o.GetName(),
	}
}
