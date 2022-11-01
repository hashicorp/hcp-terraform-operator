package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (w *Workspace) NeedToAddFinalizer(finalizer string) bool {
	return w.ObjectMeta.DeletionTimestamp.IsZero() && !controllerutil.ContainsFinalizer(w, finalizer)
}

func (w *Workspace) IsDeletionCandidate(finalizer string) bool {
	return !w.ObjectMeta.DeletionTimestamp.IsZero() && controllerutil.ContainsFinalizer(w, finalizer)
}

func (w *Workspace) IsCreationCandidate() bool {
	return w.Status.WorkspaceID == ""
}
