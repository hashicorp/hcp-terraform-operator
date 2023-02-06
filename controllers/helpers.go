// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// RETURNS
func doNotRequeue() (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func requeueAfter(duration time.Duration) (reconcile.Result, error) {
	return reconcile.Result{Requeue: true, RequeueAfter: duration}, nil
}

func requeueOnErr(err error) (reconcile.Result, error) {
	return reconcile.Result{}, err
}

func pointerOf[A any](a A) *A {
	return &a
}
