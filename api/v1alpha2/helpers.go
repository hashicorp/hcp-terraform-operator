// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

func pointerOf[A any](a A) *A {
	return &a
}
