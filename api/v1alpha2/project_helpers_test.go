// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	projectFinalizer = "project.app.terraform.io/finalizer"
)

func TestNeedToAddFinalizer(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		project  Project
		expected bool
	}{
		"HasFinalizerDeletionTimestamp": {
			Project{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{projectFinalizer},
				},
			},
			false,
		},
		"HasFinalizerNoDeletionTimestamp": {
			Project{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
					Finalizers:        []string{projectFinalizer},
				},
			},
			false,
		},
		"DoesNotHaveFinalizerDeletionTimestamp": {
			Project{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{},
				},
			},
			false,
		},
		"DoesNotHaveFinalizerNoDeletionTimestamp": {
			Project{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
					Finalizers:        []string{},
				},
			},
			true,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			out := c.project.NeedToAddFinalizer(projectFinalizer)
			if out != c.expected {
				t.Fatalf("Error matching output and expected: %#v vs %#v", out, c.expected)
			}
		})
	}
}

func TestIsDeletionCandidate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		project  Project
		expected bool
	}{
		"HasFinalizerDeletionTimestamp": {
			Project{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{projectFinalizer},
				},
			},
			true,
		},
		"HasFinalizerNoDeletionTimestamp": {
			Project{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
					Finalizers:        []string{projectFinalizer},
				},
			},
			false,
		},
		"DoesNotHaveFinalizerDeletionTimestamp": {
			Project{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Finalizers:        []string{},
				},
			},
			false,
		},
		"DoesNotHaveFinalizerNoDeletionTimestamp": {
			Project{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp: nil,
					Finalizers:        []string{},
				},
			},
			false,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			out := c.project.IsDeletionCandidate(projectFinalizer)
			if out != c.expected {
				t.Fatalf("Error matching output and expected: %#v vs %#v", out, c.expected)
			}
		})
	}
}

func TestIsCreationCandidate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		project  Project
		expected bool
	}{
		"HasID": {
			Project{Status: ProjectStatus{ID: "prj-this"}},
			false,
		},
		"DoesNotHaveID": {
			Project{Status: ProjectStatus{ID: ""}},
			true,
		},
	}

	for n, c := range cases {
		t.Run(n, func(t *testing.T) {
			out := c.project.IsCreationCandidate()
			if out != c.expected {
				t.Fatalf("Error matching output and expected: %#v vs %#v", out, c.expected)
			}
		})
	}
}
