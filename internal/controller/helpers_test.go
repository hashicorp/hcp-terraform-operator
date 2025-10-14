// Copyright (c) HashiCorp, Inc.
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
	result, err := doNotRequeue()
	assert.Nil(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestRequeueAfter(t *testing.T) {
	duration := 1 * time.Second
	result, err := requeueAfter(duration)
	assert.Nil(t, err)
	assert.Equal(t, reconcile.Result{Requeue: true, RequeueAfter: duration}, result)
}

func TestRequeueOnErr(t *testing.T) {
	result, err := requeueOnErr(fmt.Errorf(""))
	assert.NotNil(t, err)
	assert.Equal(t, reconcile.Result{}, result)
}

func TestFormatOutput(t *testing.T) {
	testCases := []struct {
		name     string
		input    *tfc.StateVersionOutput
		expected string
	}{
		{
			name: "boolean output",
			input: &tfc.StateVersionOutput{
				Type:  "boolean",
				Value: true,
			},
			expected: "true",
		},
		{
			name: "string output",
			input: &tfc.StateVersionOutput{
				Type:  "string",
				Value: "hello",
			},
			expected: "hello",
		},
		{
			name: "multiline string output",
			input: &tfc.StateVersionOutput{
				Type:  "string",
				Value: "hello\nworld",
			},
			expected: "hello\nworld",
		},
		{
			name: "number output",
			input: &tfc.StateVersionOutput{
				Type:  "number",
				Value: 162,
			},
			expected: "162",
		},
		{
			name: "list output",
			input: &tfc.StateVersionOutput{
				Type: "array",
				Value: []any{
					"one",
					2,
				},
			},
			expected: `["one",2]`,
		},
		{
			name: "map output",
			input: &tfc.StateVersionOutput{
				Type: "array",
				Value: map[string]string{
					"one": "een",
					"two": "twee",
				},
			},
			expected: `{"one":"een","two":"twee"}`,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // run test cases in parallel
			result, err := formatOutput(tc.input)
			assert.Equal(t, tc.expected, result)
			assert.Nil(t, err)
		})
	}
}

func TestFinalizerBehaviors(t *testing.T) {
	testCases := []struct {
		name           string
		hasDeletion    bool
		hasFinalizer   bool
		testFunc       func(*TestObject, string) bool
		expectedResult bool
	}{
		{
			name:           "NeedToAddFinalizer: No deletion timestamp, no finalizer",
			hasDeletion:    false,
			hasFinalizer:   false,
			testFunc:       needToAddFinalizer[*TestObject],
			expectedResult: true,
		},
		{
			name:           "NeedToAddFinalizer: No deletion timestamp, has finalizer",
			hasDeletion:    false,
			hasFinalizer:   true,
			testFunc:       needToAddFinalizer[*TestObject],
			expectedResult: false,
		},
		{
			name:           "NeedToAddFinalizer: Has deletion timestamp, no finalizer",
			hasDeletion:    true,
			hasFinalizer:   false,
			testFunc:       needToAddFinalizer[*TestObject],
			expectedResult: false,
		},
		{
			name:           "NeedToAddFinalizer: Has deletion timestamp, has finalizer",
			hasDeletion:    true,
			hasFinalizer:   true,
			testFunc:       needToAddFinalizer[*TestObject],
			expectedResult: false,
		},
		{
			name:           "IsDeletionCandidate: No deletion timestamp, no finalizer",
			hasDeletion:    false,
			hasFinalizer:   false,
			testFunc:       isDeletionCandidate[*TestObject],
			expectedResult: false,
		},
		{
			name:           "IsDeletionCandidate: No deletion timestamp, has finalizer",
			hasDeletion:    false,
			hasFinalizer:   true,
			testFunc:       isDeletionCandidate[*TestObject],
			expectedResult: false,
		},
		{
			name:           "IsDeletionCandidate: Has deletion timestamp, no finalizer",
			hasDeletion:    true,
			hasFinalizer:   false,
			testFunc:       isDeletionCandidate[*TestObject],
			expectedResult: false,
		},
		{
			name:           "IsDeletionCandidate: Has deletion timestamp, has finalizer",
			hasDeletion:    true,
			hasFinalizer:   true,
			testFunc:       isDeletionCandidate[*TestObject],
			expectedResult: true,
		},
	}

	testFinalizer := "test.app.terraform.io/finalizer"

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			o := TestObject{}
			if tc.hasDeletion {
				o.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			}
			if tc.hasFinalizer {
				o.ObjectMeta.Finalizers = []string{testFinalizer}
			}

			result := tc.testFunc(&o, testFinalizer)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestMatchWildcardName(t *testing.T) {
	testCases := []struct {
		name     string
		wildcard string
		str      string
		expected bool
	}{
		{
			name:     "match prefix",
			wildcard: "*-terraform-workspace",
			str:      "hcp-terraform-workspace",
			expected: true,
		},
		{
			name:     "match suffix",
			wildcard: "hcp-terraform-*",
			str:      "hcp-terraform-workspace",
			expected: true,
		},
		{
			name:     "match prefix and suffix",
			wildcard: "*-terraform-*",
			str:      "hcp-terraform-workspace",
			expected: true,
		},
		{
			name:     "match no prefix and no suffix",
			wildcard: "hcp-terraform-workspace",
			str:      "hcp-terraform-workspace",
			expected: true,
		},
		{
			name:     "does not match prefix",
			wildcard: "*-terraform-workspace",
			str:      "hcp-tf-workspace",
			expected: false,
		},
		{
			name:     "does not match suffix",
			wildcard: "hcp-terraform-*",
			str:      "hashicorp-tf-workspace",
			expected: false,
		},
		{
			name:     "does not match prefix and suffix",
			wildcard: "*-terraform-*",
			str:      "hcp-tf-workspace",
			expected: false,
		},
		{
			name:     "does not match no prefix and no suffix",
			wildcard: "hcp-terraform-workspace",
			str:      "hcp-tf-workspace",
			expected: false,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := matchWildcardName(tc.wildcard, tc.str)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestValidateTFEVersion(t *testing.T) {
	testCases := []struct {
		name          string
		version       string
		expectedValue bool
		expectError   bool
	}{
		{
			name:          "Valid TFE version",
			version:       "v202502-1",
			expectedValue: true,
			expectError:   false,
		},
		{
			name:          "Invalid TFE version",
			version:       "202502-1",
			expectedValue: false,
			expectError:   true,
		},
		{
			name:          "Empty TFE version",
			version:       "",
			expectedValue: true,
			expectError:   false,
		},
		{
			name:          "New valid TFE version",
			version:       "1.0.0",
			expectedValue: true,
			expectError:   false,
		},
		{
			name:          "New valid TFE version 2",
			version:       "v1.0.1",
			expectedValue: true,
			expectError:   false,
		},
		{
			name:          "New invalid TFE version",
			version:       "1.0",
			expectedValue: false,
			expectError:   true,
		},
		{
			name:          "New invalid TFE version 2",
			version:       "v1.0",
			expectedValue: false,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v, err := validateTFEVersion(tc.version)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedValue, v)
			}
		})
	}
}
