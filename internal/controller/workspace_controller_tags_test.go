// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTagDifference(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		leftTags  map[string]bool
		rightTags map[string]bool
		expect    []string
	}{
		"TagsDifference": {
			leftTags: map[string]bool{
				"A": true,
				"B": true,
			},
			rightTags: map[string]bool{
				"B": true,
			},
			expect: []string{
				"A",
			},
		},
		"TagsUnintersectioned": {
			leftTags: map[string]bool{
				"A": true,
				"B": true,
			},
			rightTags: map[string]bool{
				"C": true,
				"D": true,
			},
			expect: []string{
				"A",
				"B",
			},
		},
		"TagsDifferenceWhenRightTagsIsEmpty": {
			leftTags: map[string]bool{
				"A": true,
				"B": true,
			},
			rightTags: make(map[string]bool),
			expect: []string{
				"A",
				"B",
			},
		},
		"TagsDifferenceWhenLeftTagsIsEmpty": {
			leftTags: map[string]bool{},
			rightTags: map[string]bool{
				"A": true,
				"B": true,
			},
			expect: []string{},
		},
		"TagsEqual": {
			leftTags: map[string]bool{
				"A": true,
				"B": true,
			},
			rightTags: map[string]bool{
				"B": true,
				"A": true,
			},
			expect: []string{},
		},
		"TagsEqualEmpty": {
			leftTags:  map[string]bool{},
			rightTags: map[string]bool{},
			expect:    []string{},
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			d := tagDifference(c.leftTags, c.rightTags)
			assert.ElementsMatch(t, c.expect, d)
		})
	}
}
