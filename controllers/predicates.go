// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func handlePredicates() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Update event has no old object to update.
			if e.ObjectOld == nil {
				return false
			}

			// Update event has no new object to update.
			if e.ObjectNew == nil {
				return false
			}

			// Generation of an object changes when .spec has been updated.
			// If Generations of new and old objects are not equal the object has be to reconcile.
			if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
				return true
			}

			// ResourceVersion of an object changes when any part of the object has been updated.
			// If ResourceVersions of new and old objects are equal this is a periodic reconciliation.
			if e.ObjectOld.GetResourceVersion() == e.ObjectNew.GetResourceVersion() {
				return true
			}

			// Do not call reconciliation in all other cases
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
}
