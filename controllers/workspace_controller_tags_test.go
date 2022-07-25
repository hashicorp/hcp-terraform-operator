package controllers

import (
	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TODO these are unit tests, lets just do these as vanilla Go tests
var _ = Describe("Helpers", func() {
	Context("Tags", func() {
		It("returns difference between tags sets", func() {
			leftTags := map[string]bool{
				"A": true,
				"B": true,
			}
			rightTags := map[string]bool{
				"B": true,
			}
			expect := []*tfc.Tag{
				{Name: "A"},
			}
			d := tagDifference(leftTags, rightTags)

			Expect(d).Should(ConsistOf(expect))
		})
		It("returns leftTags as difference between unintersectioned tags sets", func() {
			leftTags := map[string]bool{
				"A": true,
				"B": true,
			}
			rightTags := map[string]bool{
				"C": true,
				"D": true,
			}
			expect := []*tfc.Tag{
				{Name: "A"},
				{Name: "B"},
			}
			d := tagDifference(leftTags, rightTags)
			Expect(d).Should(Equal(expect))
		})
		It("returns leftTags as difference between tags sets when rightTags set is empty", func() {
			leftTags := map[string]bool{
				"A": true,
				"B": true,
			}
			rightTags := make(map[string]bool)
			expect := []*tfc.Tag{
				{Name: "A"},
				{Name: "B"},
			}
			d := tagDifference(leftTags, rightTags)
			Expect(d).Should(Equal(expect))
		})
		It("returns no difference between tags sets when leftTags set is empty", func() {
			leftTags := map[string]bool{}
			rightTags := map[string]bool{
				"A": true,
				"B": true,
			}
			var expect []*tfc.Tag
			d := tagDifference(leftTags, rightTags)
			Expect(d).Should(Equal(expect))
		})
		It("returns no difference between equal tags sets", func() {
			leftTags := map[string]bool{
				"A": true,
				"B": true,
			}
			rightTags := map[string]bool{
				"B": true,
				"A": true,
			}
			var expect []*tfc.Tag
			d := tagDifference(leftTags, rightTags)
			Expect(d).Should(Equal(expect))
		})
		It("returns no difference between empty tags sets", func() {
			leftTags := map[string]bool{}
			rightTags := map[string]bool{}
			var expect []*tfc.Tag
			d := tagDifference(leftTags, rightTags)
			Expect(d).Should(Equal(expect))
		})
	})
})
