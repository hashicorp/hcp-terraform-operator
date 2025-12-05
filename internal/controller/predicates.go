// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			if e.Object.GetDeletionTimestamp() != nil {
				return deletionTimestampPredicate(e.Object)
			}

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

			if e.ObjectNew.GetDeletionTimestamp() != nil {
				return deletionTimestampPredicate(e.ObjectNew)
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

			// Continue with reconciliation if the app.terraform.io/paused annotation is set or has been removed.
			if e.ObjectNew.GetAnnotations()[annotationPaused] != "" || e.ObjectOld.GetAnnotations()[annotationPaused] != "" {
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

// workspacePredicates returns predicates that are specific for the workspace controller.
func workspacePredicates() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// TODO:
			// - Think about how to avoid double deletion timestamp checking.
			if e.ObjectNew.GetDeletionTimestamp() != nil {
				return false
			}
			// Validate if a certain annotation persists in a new object and does not match the old one.
			// In that case, it is a new or updated annotation and we need to trigger a reconciliation cycle.
			if a, ok := e.ObjectNew.GetAnnotations()[workspaceAnnotationRunNew]; ok && a == metaTrue {
				return true
			}

			// Do not call reconciliation in all other cases
			return false
		},
	}
}

func deletionTimestampPredicate(o client.Object) bool {
	finalizers := []string{
		agentPoolFinalizer,
		agentTokenFinalizer,
		moduleFinalizer,
		projectFinalizer,
		runsCollectorFinalizer,
		workspaceFinalizer,
	}

	// Do not trigger reconciliation if the object was deleted with the `foreground` option.
	// Let Kubernetes first delete all dependent objects and only then the operator proceeds with the deletion of the parent object.
	// This helps to avoid the situation when the operator triggers deletion twice.
	// The second deletion get triggered when Kubernetes GC removes the `foregroundDeletion` finalizer.
	if controllerutil.ContainsFinalizer(o, metav1.FinalizerDeleteDependents) {
		return false
	}

	// Trigger reconciliation if the object was deleted and contains one of the operator finalizers.
	for _, finalizer := range finalizers {
		if controllerutil.ContainsFinalizer(o, finalizer) {
			return true
		}
	}

	// Do not trigger reconciliation if the object was deleted and does not contain any of the operator finalizers.
	return false
}
