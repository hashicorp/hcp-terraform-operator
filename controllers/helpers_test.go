// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
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
})
