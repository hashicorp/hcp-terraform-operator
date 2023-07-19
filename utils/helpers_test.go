// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package utils

import (
	"reflect"
	"testing"
)

func TestPointerOf(t *testing.T) {
	s := "this"
	sp := PointerOf(s)
	if s != *sp {
		t.Error("Failed to get string pointer")
	}

	i := int(1984)
	ip := PointerOf(i)
	if i != *ip {
		t.Error("Failed to get int pointer")
	}

	i64 := int64(1984)
	i64p := PointerOf(i64)
	if i64 != *i64p {
		t.Error("Failed to get int64 pointer")
	}
}

func TestRemoveFromSlice(t *testing.T) {
	input := [][]any{
		{1, 2, 3},
		{"a", "b", "c"},
	}
	want := [][]any{
		{1, 3},
		{"a", "c"},
	}
	for i, s := range input {
		r := RemoveFromSlice(s, 1)
		if !reflect.DeepEqual(want[i], r) {
			t.Errorf("Failed to remove an element from slice %d", i)
		}
	}
}
