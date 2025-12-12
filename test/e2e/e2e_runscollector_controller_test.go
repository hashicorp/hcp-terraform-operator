// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
	"github.com/hashicorp/hcp-terraform-operator/internal/controller"
)

var _ = Describe("Runs Collector сontroller", Ordered, func() {
	var (
		instance       *appv1alpha2.RunsCollector
		namespacedName = newNamespacedName()
	)

	BeforeAll(func() {
		if os.Getenv("USE_EXISTING_CLUSTER") == "true" {
			Skip("Does not run on an existing cluster")
		}
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		instance = &appv1alpha2.RunsCollector{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.terraform.io/v1alpha2",
				Kind:       "RunsCollector",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              namespacedName.Name,
				Namespace:         namespacedName.Namespace,
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
			Spec: appv1alpha2.RunsCollectorSpec{
				Organization: organization,
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
				AgentPool: &appv1alpha2.AgentPoolRef{},
			},
			Status: appv1alpha2.RunsCollectorStatus{},
		}
		// Register metrics
		controller.MetricRuns.WithLabelValues(
			"pink_panther",
			"apool-pp1963",
			"pink-shadow",
		).Set(float64(162))
		controller.MetricRunsTotal.WithLabelValues(
			"apool-pp1963",
			"pink-shadow",
		).Set(float64(2134))
	})

	AfterEach(func() {
		Eventually(func() bool {
			err := k8sClient.Delete(ctx, instance)
			return kerrors.IsNotFound(err) || err == nil
		}).Should(BeTrue())

		Eventually(func() bool {
			err := k8sClient.Get(ctx, namespacedName, instance)
			return kerrors.IsNotFound(err)
		}).Should(BeTrue())
	})

	Context("When reconciling a resource", func() {
		It("It registers metrics", func() {
			metrics := readMetrics()

			Expect(strings.Contains(metrics, "hcp_tf_runs")).To(BeTrue())
			Expect(strings.Contains(metrics, "hcp_tf_runs_total")).To(BeTrue())
		})
	})
})

func readMetrics() string {
	resp, err := http.Get("http://" + metricsBindAddress + "/metrics")
	Expect(err).Should(Succeed())
	Expect(resp.StatusCode).Should(Equal(200))
	body, err := io.ReadAll(resp.Body)
	Expect(err).Should(Succeed())

	return string(body)
}
