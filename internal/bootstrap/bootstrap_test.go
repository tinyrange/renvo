//go:build !renvo

package bootstrap

import (
	"testing"

	"renvo.dev/internal/driver"
)

func TestBootstrapBackendOverrideIsConsumed(t *testing.T) {
	args, backend := bootstrapArgs([]string{"renvo-bootstrap", "-bootstrap-backend", "/tmp/backend", "test", "./pkg"}, nil)
	if len(args) != 3 || args[1] != "test" || args[2] != "./pkg" {
		t.Fatalf("bootstrap args = %#v", args)
	}
	command, ok := backend.(driver.CommandBackend)
	if !ok || command.Path != "/tmp/backend" {
		t.Fatalf("bootstrap backend = %#v", backend)
	}
}

func TestBootstrapPreservesDefaultBackend(t *testing.T) {
	want := driver.CommandBackend{Path: "/tmp/default"}
	args, backend := bootstrapArgs([]string{"renvo-bootstrap", "--help"}, want)
	if len(args) != 2 || args[1] != "--help" {
		t.Fatalf("bootstrap args = %#v", args)
	}
	command, ok := backend.(driver.CommandBackend)
	if !ok || command.Path != want.Path {
		t.Fatalf("bootstrap backend = %#v", backend)
	}
}
