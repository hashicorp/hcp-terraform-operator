package controllers

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	tfc "github.com/hashicorp/go-tfe"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cancel context.CancelFunc
var ctx context.Context
var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var tfClient *tfc.Client

var organization = os.Getenv("TFC_ORG")
var terraformToken = os.Getenv("TFC_TOKEN")

var syncPeriod = 30 * time.Second

var secretKey = "token"
var namespacedName = types.NamespacedName{
	Name:      "this",
	Namespace: "default",
}

func TestControllersAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	suiteConfig, reporterConfig := GinkgoConfiguration()
	reporterConfig.NoColor = true
	reporterConfig.Succinct = true
	reporterConfig.SlowSpecThreshold = 60 * time.Second

	RunSpecs(t, "Controllers Suite", suiteConfig, reporterConfig)
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = appv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).ToNot(HaveOccurred())

	err = appv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	if organization == "" {
		Fail("Environment variable TFC_ORG is either not set or empty but required")
	}
	if terraformToken == "" {
		Fail("Environment variable TFC_TOKEN is either not set or empty but required")
	}
	// Terraform Cloud Client
	tfClient, err = tfc.NewClient(&tfc.Config{Token: os.Getenv("TFC_TOKEN")})
	Expect(err).Should(Succeed())
	Expect(tfClient).ToNot(BeNil())

	// Kubernetes Client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	// Kubernetes Manager
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:     scheme.Scheme,
		SyncPeriod: &syncPeriod,
		Controller: v1alpha1.ControllerConfigurationSpec{
			GroupKindConcurrency: map[string]int{
				"Workspace.app.terraform.io": 5,
				"Module.app.terraform.io":    5,
			},
		},
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&WorkspaceReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("WorkspaceController"),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&ModuleReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		Recorder: k8sManager.GetEventRecorderFor("ModuleController"),
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	// Create a secret object with a TFC token that will be used by the controller
	err = k8sClient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretKey: []byte(terraformToken),
		},
	})
	Expect(err).ToNot(HaveOccurred(), "failed to create a token secret")
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
