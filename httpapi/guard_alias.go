package httpapi

import "github.com/swetjen/virtuous/guard"

// Guard carries auth metadata and middleware for a route.
type Guard = guard.Guard

// GuardSpec describes how to inject auth for a route.
type GuardSpec = guard.Spec
