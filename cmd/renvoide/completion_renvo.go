//go:build renvo

package main

import "renvo.dev/internal/driver"

func completionSourceFS() driver.SourceFS { return driver.RenvoFS{} }

func completionStdRoot(env []string) string {
	if root := completionEnv(env, "RENVO_STDROOT"); root != "" {
		return root
	}
	return "/std"
}

func completionModuleCache(env []string) string { return completionEnv(env, "RENVO_MODCACHE") }
