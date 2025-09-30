// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"time"
)

var (
	AgentPoolSyncPeriod     time.Duration
	AgentTokenSyncPeriod    time.Duration
	ModuleSyncPeriod        time.Duration
	ProjectSyncPeriod       time.Duration
	RunsCollectorSyncPeriod time.Duration
	WorkspaceSyncPeriod     time.Duration
)
