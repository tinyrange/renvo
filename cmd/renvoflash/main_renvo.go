//go:build renvo && (linux || darwin)

package main

import "renvo.dev/internal/espflash"

func appMain(args []string, env []string) int { return espflash.Run(args, env) }
