//go:build rtg

package main

import "j5.nz/rtg/rtg/internal/driver"

func completionSourceFS() driver.SourceFS { return driver.RTGFS{} }

func completionStdRoot(env []string) string {
	if root := completionEnv(env, "RTG_STDROOT"); root != "" {
		return root
	}
	return "/std"
}

func completionModuleCache(env []string) string { return completionEnv(env, "RTG_MODCACHE") }
