package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (m *Module) NeedToAddFinalizer(finalizer string) bool {
	return m.ObjectMeta.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(m, finalizer)
}

func (m *Module) IsDeletionCandidate(finalizer string) bool {
	return !m.ObjectMeta.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(m, finalizer)
}
