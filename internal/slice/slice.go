// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package slice

func RemoveFromSlice[A any](slice []A, i int) []A {
	return append(slice[:i], slice[i+1:]...)
}
