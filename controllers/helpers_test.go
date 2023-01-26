// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Helpers", func() {
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
})
