//go:build !rtg

package main

import "j5.nz/rtg/rtg/internal/driver"

func completionSourceFS() driver.SourceFS       { return driver.OSFS{} }
func completionStdRoot(env []string) string     { return driver.StdRootFromEnv(env) }
func completionModuleCache(env []string) string { return driver.EnvValue(env, driver.ModuleCacheEnv) }
