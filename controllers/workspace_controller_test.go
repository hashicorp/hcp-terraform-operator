package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, Serial, func() {
	var instance *appv1alpha2.Workspace

	BeforeEach(func() {
		instance = &appv1alpha2.Workspace{
			ObjectMeta: metav1.ObjectMeta{
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
		}
	})

	Context("Is Deletion Candidate", func() {
		It("has no DeletionTimestamp and no Finalizers", func() {
			Expect(isDeletionCandidate(instance)).Should(BeFalse())
		})
		It("has a DeletionTimestamp and no Finalizers", func() {
			instance.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(isDeletionCandidate(instance)).Should(BeFalse())
		})
		It("has no DeletionTimestamp and a Finalizer", func() {
			instance.ObjectMeta.Finalizers = []string{workspaceFinalizer}
			Expect(isDeletionCandidate(instance)).Should(BeFalse())
		})
		It("has a DeletionTimestamp and a Finalizer", func() {
			instance.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			instance.ObjectMeta.Finalizers = []string{workspaceFinalizer}
			Expect(isDeletionCandidate(instance)).Should(BeTrue())
		})
	})

	Context("Is Need To Add Finalizer", func() {
		It("has no DeletionTimestamp and no Finalizers", func() {
			Expect(needToAddFinalizer(instance)).Should(BeTrue())
		})
		It("has a DeletionTimestamp and no Finalizers", func() {
			instance.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			Expect(needToAddFinalizer(instance)).Should(BeFalse())
		})
		It("has no DeletionTimestamp and a Finalizer", func() {
			instance.ObjectMeta.Finalizers = []string{workspaceFinalizer}
			Expect(needToAddFinalizer(instance)).Should(BeFalse())
		})
		It("has a DeletionTimestamp and a Finalizer", func() {
			instance.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			instance.ObjectMeta.Finalizers = []string{workspaceFinalizer}
			Expect(needToAddFinalizer(instance)).Should(BeFalse())
		})
	})
})
