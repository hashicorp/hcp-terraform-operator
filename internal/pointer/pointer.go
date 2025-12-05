// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package pointer

func PointerOf[A any](a A) *A {
	return &a
}
