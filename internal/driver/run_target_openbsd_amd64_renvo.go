//go:build renvo && openbsd && amd64

package driver

func renvoRunTarget() string { return "openbsd/amd64" }
func renvoRunTargetID() int  { return 13 }
