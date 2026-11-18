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

	"github.com/hashicorp/go-tfe/v2/api/models"
	"github.com/microsoft/kiota-abstractions-go/serialization"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/hashicorp/hcp-terraform-operator/internal/pointer"
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
func formatOutput(o models.StateVersionOutputsable) (string, error) {
	var value serialization.UntypedNodeable
	if o != nil && o.GetAttributes() != nil {
		value = o.GetAttributes().GetValue()
	}
	return formatOutputValue(untypedNodeValue(value))
}

// formatOutputValue formats a native Go representation of an output value (as returned by untypedNodeValue).
func formatOutputValue(value any) (string, error) {
	switch x := value.(type) {
	case bool:
		return strconv.FormatBool(x), nil
	case float64:
		return fmt.Sprint(x), nil
	case string:
		return x, nil
	default:
		b, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

// untypedNodeValue converts a Kiota untyped node into its native Go representation:
// bool, float64 (for all numbers), string, nil, []any and map[string]any.
func untypedNodeValue(n serialization.UntypedNodeable) any {
	switch x := n.(type) {
	case nil:
		return nil
	case *serialization.UntypedBoolean:
		return pointer.ValueOf(x.GetValue())
	case *serialization.UntypedString:
		return pointer.ValueOf(x.GetValue())
	case *serialization.UntypedInteger:
		return float64(pointer.ValueOf(x.GetValue()))
	case *serialization.UntypedLong:
		return float64(pointer.ValueOf(x.GetValue()))
	case *serialization.UntypedFloat:
		return float64(pointer.ValueOf(x.GetValue()))
	case *serialization.UntypedDouble:
		return pointer.ValueOf(x.GetValue())
	case *serialization.UntypedNull:
		return nil
	case *serialization.UntypedArray:
		items := x.GetValue()
		a := make([]any, len(items))
		for i, item := range items {
			a[i] = untypedNodeValue(item)
		}
		return a
	case *serialization.UntypedObject:
		props := x.GetValue()
		m := make(map[string]any, len(props))
		for k, v := range props {
			m[k] = untypedNodeValue(v)
		}
		return m
	default:
		if g, ok := x.(interface{ GetValue() any }); ok {
			return g.GetValue()
		}
		return nil
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

// useRunsEndpoint determines whether to use the Runs endpoint(available since v202409-1) based on the TFE version.
func useRunsEndpoint(version string) (bool, error) {
	// For versions 1.0.0 and 1.0.1 version string will be empty.
	if version == "" {
		return true, nil
	}

	// Check for the Calendar Version format vYYYYMM-N (e.g., v202310-1).
	re := regexp.MustCompile(`^v([0-9]{6})-([0-9]{1})$`)
	matches := re.FindStringSubmatch(version)
	if len(matches) == 3 {
		calVer, err := strconv.Atoi(matches[1] + matches[2])
		if err != nil {
			return false, err
		}
		if calVer >= 2024091 {
			return true, nil
		}
	}

	return false, fmt.Errorf("malformed TFE version %s", version)
}

// nextPage returns the next page number of a paginated response using its 'meta' object,
// or zero if there are no more pages.
func nextPage(meta any) int32 {
	switch m := meta.(type) {
	case interface{ GetPagination() models.Paginationable }:
		if p := m.GetPagination(); p != nil {
			return pointer.ValueOf(p.GetNextPage())
		}
	case interface {
		GetPagination() models.PaginationWithoutTotalsable
	}:
		if p := m.GetPagination(); p != nil {
			return pointer.ValueOf(p.GetNextPage())
		}
	}
	return 0
}

// runAttributesStatus returns the run status of the given run attributes.
// A status that the v2 enum cannot represent (e.g. `post_plan_awaiting_decision`) is
// deserialized as nil; runStatusTransport preserves it as an additional attribute.
func runAttributesStatus(a models.Runs_attributesable) string {
	if a == nil {
		return ""
	}
	if s := a.GetStatus(); s != nil {
		return s.String()
	}
	if raw, ok := a.GetAdditionalData()[rawRunStatusKey].(*string); ok && raw != nil {
		return *raw
	}
	return ""
}
