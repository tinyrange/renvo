//go:build !renvo

package main

import "renvo.dev/internal/driver"

func completionSourceFS() driver.SourceFS       { return driver.OSFS{} }
func completionStdRoot(env []string) string     { return driver.StdRootFromEnv(env) }
func completionModuleCache(env []string) string { return driver.EnvValue(env, driver.ModuleCacheEnv) }
