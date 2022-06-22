package controllers

import "time"

// SHARED CONSTANTS
const (
	requeueInterval = 30 * time.Second
)

// WORKSPACE CONTROLLER'S CONSTANTS
const (
	workspaceFinalizer = "workspace.app.terraform.io/finalizer"
)
