// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"testing"

	tfc "github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
)

func TestTagDifference(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		leftTags  map[string]bool
		rightTags map[string]bool
		expect    []*tfc.Tag
	}{
		"TagsDifference": {
			leftTags: map[string]bool{
				"A": true,
				"B": true,
			},
			rightTags: map[string]bool{
				"B": true,
			},
			expect: []*tfc.Tag{
				{Name: "A"},
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
			expect: []*tfc.Tag{
				{Name: "A"},
				{Name: "B"},
			},
		},
		"TagsDifferenceWhenRightTagsIsEmpty": {
			leftTags: map[string]bool{
				"A": true,
				"B": true,
			},
			rightTags: make(map[string]bool),
			expect: []*tfc.Tag{
				{Name: "A"},
				{Name: "B"},
			},
		},
		"TagsDifferenceWhenLeftTagsIsEmpty": {
			leftTags: map[string]bool{},
			rightTags: map[string]bool{
				"A": true,
				"B": true,
			},
			expect: []*tfc.Tag{},
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
			expect: []*tfc.Tag{},
		},
		"TagsEqualEmpty": {
			leftTags:  map[string]bool{},
			rightTags: map[string]bool{},
			expect:    []*tfc.Tag{},
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			d := tagDifference(c.leftTags, c.rightTags)
			assert.ElementsMatch(t, c.expect, d)
		})
	}
}
