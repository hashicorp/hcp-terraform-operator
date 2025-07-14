// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"fmt"
)

func (r *AgentTokenReconciler) deleteAgentToken(ctx context.Context, t *agentTokenInstance) error {
	t.log.Info("Reconcile Agent Token", "msg", fmt.Sprintf("deletion policy is %s", t.instance.Spec.DeletionPolicy))

	return r.removeFinalizer(ctx, t)
}
