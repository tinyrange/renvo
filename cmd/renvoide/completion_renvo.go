//go:build renvo

package main

import "renvo.dev/internal/driver"

func completionReadDir(path string) ([]driver.DirEntry, bool) {
	return (driver.RenvoFS{}).ReadDir(path)
}

func completionReadFile(path string) ([]byte, bool) { return (driver.RenvoFS{}).ReadFile(path) }

func completionStdRoot(env []string) string {
	if root := completionEnv(env, "RENVO_STDROOT"); root != "" {
		return root
	}
	return "/std"
}

func completionModuleCache(env []string) string { return completionEnv(env, "RENVO_MODCACHE") }
