// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	tfc "github.com/hashicorp/go-tfe/v2"
	"github.com/hashicorp/go-tfe/v2/api/models"
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
	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
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
	insecure := false
	if v, ok := os.LookupEnv("TFC_TLS_SKIP_VERIFY"); ok {
		insecure, err = strconv.ParseBool(v)
		if err != nil {
			Fail(fmt.Sprintf("Cannot convert value of TFC_TLS_SKIP_VERIFY into a bool: %v", err))
		}
	}
	fmt.Fprintf(GinkgoWriter, "TFC_TLS_SKIP_VERIFY: %v", insecure)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure}
	tfcConfig := &tfc.Config{
		Token:         os.Getenv("TFC_TOKEN"),
		HTTPTransport: transport,
		Headers: http.Header{
			"User-Agent": []string{version.UserAgent},
		},
	}
	// The v2 default config does not read TFE_ADDRESS from the environment.
	if v, ok := os.LookupEnv("TFE_ADDRESS"); ok && v != "" {
		tfcConfig.Address = v
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

// isEnterprise reports whether the remote is Terraform Enterprise (not HCP Terraform).
// go-tfe v2 does not expose this, so it is read from the headers of the ping endpoint.
func isEnterprise() bool {
	GinkgoHelper()
	resp, err := tfClient.GetStream(ctx, "/api/v2/ping", http.Header{"Accept": []string{tfc.ContentTypeJSONAPI}})
	Expect(err).Should(Succeed())
	defer resp.Body.Close() //nolint:errcheck
	return resp.Header.Get("TFP-AppName") != "HCP Terraform"
}

// nextPage returns the next page number of a paginated response using its 'meta' object,
// or zero if there are no more pages.
func nextPage(meta any) int32 {
	switch m := meta.(type) {
	case interface{ GetPagination() models.Paginationable }:
		if p := m.GetPagination(); p != nil {
			return pointer.ValueOf(p.GetNextPage())
		}
	case interface {
		GetPagination() models.PaginationWithoutTotalsable
	}:
		if p := m.GetPagination(); p != nil {
			return pointer.ValueOf(p.GetNextPage())
		}
	}
	return 0
}

// patchWorkspaceRaw patches a workspace with a hand-made JSON:API payload.
// The generated models cannot express explicit JSON nulls.
func patchWorkspaceRaw(id string, attributes map[string]any) {
	GinkgoHelper()
	builder := tfClient.API.Workspaces().ByWorkspace_id(id)
	ri, err := builder.ToPatchRequestInformation(ctx, models.NewWorkspacesEnvelope(), nil)
	Expect(err).Should(Succeed())
	b, err := json.Marshal(map[string]any{
		"data": map[string]any{
			"type":       "workspaces",
			"attributes": attributes,
		},
	})
	Expect(err).Should(Succeed())
	ri.SetStreamContentAndContentType(b, "application/vnd.api+json")
	res, err := builder.RequestAdapter.Send(ctx, ri, models.CreateWorkspacesEnvelopeFromDiscriminatorValue, nil)
	Expect(err).Should(Succeed())
	Expect(res).ShouldNot(BeNil())
}

// tarGz packs the regular files of a directory into an in-memory tar.gz archive.
func tarGz(dir string) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	tw := tar.NewWriter(gz)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close() //nolint:errcheck
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf, nil
}

// uploadConfigurationVersion uploads the content of a directory to a given configuration version upload URL.
// go-tfe v2 does not provide an upload helper.
func uploadConfigurationVersion(uploadURL, dir string) error {
	archive, err := tarGz(dir)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, archive)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	insecure := false
	if v, ok := os.LookupEnv("TFC_TLS_SKIP_VERIFY"); ok {
		insecure, _ = strconv.ParseBool(v)
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure} //nolint:gosec
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("configuration version upload failed with status %d", resp.StatusCode)
	}
	return nil
}
