//go:build !renvo && !renvo_native_backend

package main

import (
	"renvo.dev/internal/backendjit"
	"renvo.dev/internal/driver"
)

// The ordinary Go frontend keeps VM32 as its only compiled-in backend and
// promotes native compilers on demand.
func checkoutBackend(stdRoot string, cacheDir string) driver.Backend {
	return checkoutBackendMux{builtin: backendjit.NewBuiltin(stdRoot, cacheDir)}
}
