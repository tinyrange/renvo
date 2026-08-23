//go:build !renvo && renvo_native_backend

package main

import (
	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
)

// renvo_native_backend deliberately trades a larger Go executable for an
// in-process backend with no promotion step or persistent backend cache.
func checkoutBackend(_ string, _ string) driver.Backend {
	return checkoutBackendMux{builtin: backendcompiled.Backend{}}
}
