// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"fmt"
	"testing"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type TestObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func (in *TestObject) DeepCopyObject() runtime.Object {
	return nil
}

func TestDoNotRequeue(t *testing.T) {
	t.Parallel()
	result, err := doNotRequeue()
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestRequeueAfter(t *testing.T) {
	t.Parallel()
	duration := 1 * time.Second
	result, err := requeueAfter(duration)
	assert.NoError(t, err)
	assert.Equal(t, reconcile.Result{Requeue: true, RequeueAfter: duration}, result)
}

func TestRequeueOnErr(t *testing.T) {
	t.Parallel()
	result, err := requeueOnErr(fmt.Errorf(""))
	assert.Error(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestFormatOutput(t *testing.T) {
	t.Parallel()
	successCases := map[string]struct {
		input    *tfc.StateVersionOutput
		expected string
	}{
		"Boolean": {
			input: &tfc.StateVersionOutput{
				Type:  "boolean",
				Value: true,
			},
			expected: "true",
		},
		"String": {
			input: &tfc.StateVersionOutput{
				Type:  "string",
				Value: "hello",
			},
			expected: "hello",
		},
		"MultilineString": {
			input: &tfc.StateVersionOutput{
				Type:  "string",
				Value: "hello\nworld",
			},
			expected: "hello\nworld",
		},
		"Number": {
			input: &tfc.StateVersionOutput{
				Type:  "number",
				Value: 162,
			},
			expected: "162",
		},
		"List": {
			input: &tfc.StateVersionOutput{
				Type: "array",
				Value: []any{
					"one",
					2,
				},
			},
			expected: `["one",2]`,
		},
		"Map": {
			input: &tfc.StateVersionOutput{
				Type: "map",
				Value: map[string]string{
					"one": "een",
					"two": "twee",
				},
			},
			expected: `{"one":"een","two":"twee"}`,
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			result, err := formatOutput(c.input)
			assert.NoError(t, err)
			assert.Equal(t, c.expected, result)
		})
	}

	errorCases := map[string]struct {
		input    *tfc.StateVersionOutput
		expected string
	}{
		"MalformedJSON": {
			input: &tfc.StateVersionOutput{
				Type: "map",
				Value: map[string]any{
					"one":  "een",
					"func": func() {},
				},
			},
			expected: "",
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			result, err := formatOutput(c.input)
			assert.Error(t, err)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestNeedToAddFinalizer(t *testing.T) {
	t.Parallel()
	testFinalizer := "test.app.terraform.io/finalizer"
	cases := map[string]struct {
		o        *TestObject
		expected bool
	}{
		"NoDeletionTimestampNoFinalizer": {
			o: &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
					Finalizers:        []string{}},
			},
			expected: true,
		},
		"NoDeletionTimestampHasFinalizer": {
			o: &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
					Finalizers:        []string{testFinalizer}},
			},
			expected: false,
		},
		"HasDeletionTimestampNoFinalizer": {
			o: &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{}},
			},
			expected: false,
		},
		"HasDeletionTimestampHasFinalizer": {
			o: &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{testFinalizer}},
			},
			expected: false,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			result := needToAddFinalizer(c.o, testFinalizer)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestIsDeletionCandidate(t *testing.T) {
	t.Parallel()
	testFinalizer := "test.app.terraform.io/finalizer"
	cases := map[string]struct {
		o        *TestObject
		expected bool
	}{
		"NoDeletionTimestampNoFinalizer": {
			o: &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
					Finalizers:        []string{}},
			},
			expected: false,
		},
		"NoDeletionTimestampHasFinalizer": {
			o: &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
					Finalizers:        []string{testFinalizer}},
			},
			expected: false,
		},
		"HasDeletionTimestampNoFinalizer": {
			o: &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{}},
			},
			expected: false,
		},
		"HasDeletionTimestampHasFinalizer": {
			o: &TestObject{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{testFinalizer}},
			},
			expected: true,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			result := isDeletionCandidate(c.o, testFinalizer)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestMatchWildcardName(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		wildcard string
		str      string
		expected bool
	}{
		"MatchPrefix": {
			wildcard: "*-terraform-workspace",
			str:      "hcp-terraform-workspace",
			expected: true,
		},
		"MatchSuffix": {
			wildcard: "hcp-terraform-*",
			str:      "hcp-terraform-workspace",
			expected: true,
		},
		"MatchPrefixAndSuffix": {
			wildcard: "*-terraform-*",
			str:      "hcp-terraform-workspace",
			expected: true,
		},
		"MatchNoPrefixNoSuffix": {
			wildcard: "hcp-terraform-workspace",
			str:      "hcp-terraform-workspace",
			expected: true,
		},
		"DoesNotMatchPrefix": {
			wildcard: "*-terraform-workspace",
			str:      "hcp-tf-workspace",
			expected: false,
		},
		"DoesNotMatchSuffix": {
			wildcard: "hcp-terraform-*",
			str:      "hashicorp-tf-workspace",
			expected: false,
		},
		"DoesNotMatchPrefixAndSuffix": {
			wildcard: "*-terraform-*",
			str:      "hcp-tf-workspace",
			expected: false,
		},
		"DeosNotMatchNoPrefixNoSuffix": {
			wildcard: "hcp-terraform-workspace",
			str:      "hcp-tf-workspace",
			expected: false,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			result := matchWildcardName(c.wildcard, c.str)
			assert.Equal(t, c.expected, result)
		})
	}
}

func TestUseRunsEndpoint(t *testing.T) {
	t.Parallel()
	successCases := map[string]struct {
		version  string
		expected bool
	}{
		"ValidTFEVersion": {
			version:  "v202502-1",
			expected: true,
		},
		"EmptyTFEVersion": {
			version:  "",
			expected: true,
		},
	}

	for n, c := range successCases {
		t.Run(n, func(t *testing.T) {
			v, err := useRunsEndpoint(c.version)
			assert.NoError(t, err)
			assert.Equal(t, c.expected, v)
		})
	}

	errorCases := map[string]struct {
		version  string
		expected bool
	}{
		"HasMissedVPrefix": {
			version:  "202502-1",
			expected: false,
		},
	}

	for n, c := range errorCases {
		t.Run(n, func(t *testing.T) {
			v, err := useRunsEndpoint(c.version)
			assert.Error(t, err)
			assert.Equal(t, c.expected, v)
		})
	}
}
