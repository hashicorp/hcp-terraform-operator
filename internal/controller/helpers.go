// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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

func validateTFEVersion(version string) (bool, error) {
	// For versions 1.0.0 and 1.0.1 version string will be empty
	if version == "" {
		return true, nil
	}

	// Check for the version format vYYYYMM-N (e.g., v202310-1)
	versionRegexp := regexp.MustCompile(`^v([0-9]{6})-([0-9]{1})$`)
	matches := versionRegexp.FindStringSubmatch(version)
	if len(matches) == 3 {
		dateVersion, err := strconv.Atoi(matches[1] + matches[2])
		if err != nil {
			return false, err
		}
		if dateVersion >= 2024091 {
			return true, nil
		}
	}

	// Check for the version format vX.Y.Z (e.g., v1.2.3) or X.Y.Z (e.g., 1.2.3)
	isASemVer, err := regexp.MatchString(`v?([0-9]+)\.([0-9]+)\.([0-9]+)`, version)
	if err != nil {
		return false, err
	}
	if isASemVer {
		return true, nil
	}

	// If the version does not match any of the expected formats, return an error
	return false, fmt.Errorf("malformed TFE version %s", version)
}
