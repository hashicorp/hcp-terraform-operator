// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func doNotRequeue() (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func requeueAfter(duration time.Duration) (reconcile.Result, error) {
	return reconcile.Result{Requeue: true, RequeueAfter: duration}, nil
}

func requeueOnErr(err error) (reconcile.Result, error) {
	return reconcile.Result{}, err
}

// formatOutput formats TFC/E output to a string or bytes to save it further in
// Kubernetes ConfigMap or Secret, respectively.
//
// Terraform supports the following types:
// - https://developer.hashicorp.com/terraform/language/expressions/types
// When the output value is `null`(special value), TFC/E does not return it.
// Thus, we do not catch it here.
func formatOutput(o *tfc.StateVersionOutput) (string, error) {
	switch x := o.Value.(type) {
	case bool:
		return strconv.FormatBool(x), nil
	case float64:
		return fmt.Sprint(x), nil
	case string:
		return x, nil
	default:
		b, err := json.Marshal(o.Value)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

type Object interface {
	client.Object
}

// needToAddFinalizer reports true when a given object doesn't contain a given finalizer and it is not marked for deletion.
// Otherwise, it reports false.
func needToAddFinalizer[T Object](o T, finalizer string) bool {
	return o.GetDeletionTimestamp().IsZero() && !controllerutil.ContainsFinalizer(o, finalizer)
}

// isDeletionCandidate reports true when a given object contains a given finalizer and it is marked for deletion.
// Otherwise, it reports false.
func isDeletionCandidate[T Object](o T, finalizer string) bool {
	return !o.GetDeletionTimestamp().IsZero() && controllerutil.ContainsFinalizer(o, finalizer)
}

// configMapKeyRef fetches a given key name from a given Kubernetes Config Map.
func configMapKeyRef(ctx context.Context, c client.Client, nn types.NamespacedName, key string) (string, error) {
	cm := &corev1.ConfigMap{}
	if err := c.Get(ctx, nn, cm); err != nil {
		return "", err
	}

	if k, ok := cm.Data[key]; ok {
		return k, nil
	}

	return "", fmt.Errorf("unable to find key=%q in configMap=%q namespace=%q", key, nn.Name, nn.Namespace)
}

// secretKeyRef fetches a given key name from a given Kubernetes Secret.
func secretKeyRef(ctx context.Context, c client.Client, nn types.NamespacedName, key string) (string, error) {
	secret := &corev1.Secret{}
	if err := c.Get(ctx, nn, secret); err != nil {
		return "", err
	}

	if k, ok := secret.Data[key]; ok {
		return strings.TrimSpace(string(k)), nil
	}

	return "", fmt.Errorf("unable to find key=%q in secret=%q namespace=%q", key, nn.Name, nn.Namespace)
}
