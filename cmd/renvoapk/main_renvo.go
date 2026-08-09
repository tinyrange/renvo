//go:build renvo

package main

func appMain(args []string) int {
	if len(args) == 0 {
		return run(nil)
	}
	return run(args[1:])
}
