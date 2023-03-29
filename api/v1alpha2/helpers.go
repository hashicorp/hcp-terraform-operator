// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

func PointerOf[A any](a A) *A {
	return &a
}

func RemoveFromSlice[A any](slice []A, i int) []A {
	return append(slice[:i], slice[i+1:]...)
}
