// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// genericPredicates return predicates that are common for all controllers.
// Controller specific predicates should be named <CONTROLLER-NAME>Predicates.
// For example: workspacePredicates().
func genericPredicates() predicate.Predicate {
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

			// Do not trigger reconciliation if an object was deleted with the `foreground` option.
			// Let Kubernetes first delete all dependent objects and only then the operator proceeds with the deletion of the parent object.
			// This helps to avoid the situation when the operator triggers deletion twice.
			// The second deletion get triggered when Kubernetes GC removes the `foregroundDeletion` finalizer.
			if e.ObjectNew.GetDeletionTimestamp() != nil {
				return !controllerutil.ContainsFinalizer(e.ObjectNew, metav1.FinalizerDeleteDependents)
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
