//go:build renvo && netbsd && amd64

package driver

func renvoRunTarget() string { return "netbsd/amd64" }
func renvoRunTargetID() int  { return 14 }
