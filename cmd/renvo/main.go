//go:build !renvo

package main

import (
	"os"

	"renvo.dev/internal/bootstrap"
	"renvo.dev/internal/driver"
)

func main() {
	env := os.Environ()
	if driver.EnvValue(env, driver.StdRootEnv) == "" {
		stdRoot := "./std"
		if _, err := os.Stat(stdRoot); err != nil {
			stdRoot = "../std"
		}
		env = append(env, driver.StdRootEnv+"="+stdRoot)
	}
	os.Exit(bootstrap.Run(os.Args, env, checkoutBackend()))
}

func checkoutBackend() driver.Backend {
	path := "."
	if _, err := os.Stat("compiler_main.go"); err != nil {
		path = "./backend"
	}
	return driver.CommandBackend{Path: "go", Args: []string{"run", path}}
}
