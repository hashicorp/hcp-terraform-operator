// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package pointer

func PointerOf[A any](a A) *A {
	return &a
}

// ValueOf returns the value a pointer points to, or the zero value if the pointer is nil.
func ValueOf[A any](a *A) A {
	if a == nil {
		var zero A
		return zero
	}
	return *a
}
