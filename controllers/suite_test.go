// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

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

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
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

var syncPeriod = 30 * time.Second

var secretKey = "token"
var secretNamespacedName = types.NamespacedName{
	Name:      "this",
	Namespace: metav1.NamespaceDefault,
}

var rndm = rand.New(rand.NewSource(GinkgoRandomSeed()))

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
		testEnv.CRDDirectoryPaths = []string{filepath.Join("..", "config", "crd", "bases")}
		testEnv.ErrorIfCRDPathMissing = true
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = appv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())
	//+kubebuilder:scaffold:scheme

	if organization == "" {
		Fail("Environment variable TFC_ORG is required, but either not set or empty")
	}
	if terraformToken == "" {
		Fail("Environment variable TFC_TOKEN is required, but either not set or empty")
	}
	// Terraform Cloud Client
	tfClient, err = tfc.NewClient(&tfc.Config{Token: os.Getenv("TFC_TOKEN")})
	Expect(err).ToNot(HaveOccurred())
	Expect(tfClient).ToNot(BeNil())
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
					"AgentPool.app.terraform.io": 5,
					"Module.app.terraform.io":    5,
					"Project.app.terraform.io":   5,
					"Workspace.app.terraform.io": 5,
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		err = (&AgentPoolReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("AgentPoolController"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		err = (&ModuleReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("ModuleController"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		err = (&ProjectReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorderFor("ProjectController"),
		}).SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())

		err = (&WorkspaceReconciler{
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
			secretKey: []byte(terraformToken),
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

func getNamespacedName[T Object](o T) types.NamespacedName {
	GinkgoHelper()
	return types.NamespacedName{
		Namespace: o.GetNamespace(),
		Name:      o.GetName(),
	}
}
