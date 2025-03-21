// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type TestObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func (in *TestObject) DeepCopyObject() runtime.Object {
	return nil
}

var _ = Describe("Helpers", Label("Unit"), func() {
	Context("Returns", func() {
		It("do not requeue", func() {
			result, err := doNotRequeue()
			Expect(result).To(BeEquivalentTo(reconcile.Result{}))
			Expect(err).To(BeNil())
		})
		It("requeue after", func() {
			duration := 1 * time.Second
			result, err := requeueAfter(duration)
			Expect(result).To(BeEquivalentTo(reconcile.Result{Requeue: true, RequeueAfter: duration}))
			Expect(err).To(BeNil())
		})
		It("requeue on error", func() {
			result, err := requeueOnErr(fmt.Errorf(""))
			Expect(result).To(BeEquivalentTo(reconcile.Result{}))
			Expect(err).ToNot(BeNil())
		})
	})

	Context("FormatOutput", func() {
		It("bool", func() {
			o := &tfc.StateVersionOutput{
				Type:  "boolean",
				Value: true,
			}
			e := "true"
			result, err := formatOutput(o)
			Expect(result).To(BeEquivalentTo(e))
			Expect(err).To(BeNil())
		})
		It("string", func() {
			o := &tfc.StateVersionOutput{
				Type:  "string",
				Value: "hello",
			}
			e := "hello"
			result, err := formatOutput(o)
			Expect(result).To(BeEquivalentTo(e))
			Expect(err).To(BeNil())
		})
		It("multilineString", func() {
			o := &tfc.StateVersionOutput{
				Type:  "string",
				Value: "hello\nworld",
			}
			e := "hello\nworld"
			result, err := formatOutput(o)
			Expect(result).To(BeEquivalentTo(e))
			Expect(err).To(BeNil())
		})
		It("number", func() {
			o := &tfc.StateVersionOutput{
				Type:  "number",
				Value: 162,
			}
			e := "162"
			result, err := formatOutput(o)
			Expect(result).To(BeEquivalentTo(e))
			Expect(err).To(BeNil())
		})
		It("list", func() {
			o := &tfc.StateVersionOutput{
				Type: "array",
				Value: []any{
					"one",
					2,
				},
			}
			e := `["one",2]`
			result, err := formatOutput(o)
			Expect(result).To(BeEquivalentTo(e))
			Expect(err).To(BeNil())
		})
		It("map", func() {
			o := &tfc.StateVersionOutput{
				Type: "array",
				Value: map[string]string{
					"one": "een",
					"two": "twee",
				},
			}
			e := `{"one":"een","two":"twee"}`
			result, err := formatOutput(o)
			Expect(result).To(BeEquivalentTo(e))
			Expect(err).To(BeNil())
		})
	})

	Context("NeedToAddFinalizer", func() {
		testFinalizer := "test.app.terraform.io/finalizer"
		o := TestObject{}
		It("No deletion timestamp and no finalizer", func() {
			o.ObjectMeta.DeletionTimestamp = nil
			o.ObjectMeta.Finalizers = []string{}
			Expect(needToAddFinalizer(&o, testFinalizer)).To(BeTrue())
		})
		It("No deletion timestamp and finalizer", func() {
			o.ObjectMeta.DeletionTimestamp = nil
			o.ObjectMeta.Finalizers = []string{testFinalizer}
			Expect(needToAddFinalizer(&o, testFinalizer)).To(BeFalse())
		})
		It("Deletion timestamp and no finalizer", func() {
			o.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			o.ObjectMeta.Finalizers = []string{}
			Expect(needToAddFinalizer(&o, testFinalizer)).To(BeFalse())
		})
		It("Deletion timestamp and finalizer", func() {
			o.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			o.ObjectMeta.Finalizers = []string{testFinalizer}
			Expect(needToAddFinalizer(&o, testFinalizer)).To(BeFalse())
		})
	})

	Context("IsDeletionCandidate", func() {
		testFinalizer := "test.app.terraform.io/finalizer"
		o := TestObject{}
		It("No deletion timestamp and no finalizer", func() {
			o.ObjectMeta.DeletionTimestamp = nil
			o.ObjectMeta.Finalizers = []string{}
			Expect(isDeletionCandidate(&o, testFinalizer)).To(BeFalse())
		})
		It("No deletion timestamp and finalizer", func() {
			o.ObjectMeta.DeletionTimestamp = nil
			o.ObjectMeta.Finalizers = []string{testFinalizer}
			Expect(isDeletionCandidate(&o, testFinalizer)).To(BeFalse())
		})
		It("Deletion timestamp and no finalizer", func() {
			o.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			o.ObjectMeta.Finalizers = []string{}
			Expect(isDeletionCandidate(&o, testFinalizer)).To(BeFalse())
		})
		It("Deletion timestamp and finalizer", func() {
			o.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			o.ObjectMeta.Finalizers = []string{testFinalizer}
			Expect(isDeletionCandidate(&o, testFinalizer)).To(BeTrue())
		})
	})

	Context("Match wildcard name", func() {
		// True
		It("match prefix", func() {
			result := matchWildcardName("*-terraform-workspace", "hcp-terraform-workspace")
			Expect(result).To(BeTrue())
		})
		It("match suffix", func() {
			result := matchWildcardName("hcp-terraform-*", "hcp-terraform-workspace")
			Expect(result).To(BeTrue())
		})
		It("match prefix and suffix", func() {
			result := matchWildcardName("*-terraform-*", "hcp-terraform-workspace")
			Expect(result).To(BeTrue())
		})
		It("match no prefix and no suffix", func() {
			result := matchWildcardName("hcp-terraform-workspace", "hcp-terraform-workspace")
			Expect(result).To(BeTrue())
		})
		// False
		It("does not match prefix", func() {
			result := matchWildcardName("*-terraform-workspace", "hcp-tf-workspace")
			Expect(result).To(BeFalse())
		})
		It("does not match suffix", func() {
			result := matchWildcardName("hcp-terraform-*", "hashicorp-tf-workspace")
			Expect(result).To(BeFalse())
		})
		It("does not match prefix and suffix", func() {
			result := matchWildcardName("*-terraform-*", "hcp-tf-workspace")
			Expect(result).To(BeFalse())
		})
		It("does not match no prefix and no suffix", func() {
			result := matchWildcardName("hcp-terraform-workspace", "hcp-tf-workspace")
			Expect(result).To(BeFalse())
		})
	})
})
