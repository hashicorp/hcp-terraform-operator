// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Helpers", Label("Unit"), func() {
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
