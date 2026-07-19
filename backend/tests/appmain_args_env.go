package main

func appmainHasPrefix(s string, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}

func appMain(args []string, env []string) int {
	if len(args) == 0 {
		print("missing args\n")
		return 1
	}
	for i := 0; i < len(env); i++ {
		if appmainHasPrefix(env[i], "PATH=") {
			print("PASS\n")
			return 0
		}
	}
	print("missing env\n")
	return 1
}
