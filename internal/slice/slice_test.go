// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package slice

import (
	"reflect"
	"testing"
)

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
