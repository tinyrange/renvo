package main

func preparedHasPrefix(value string, prefix string) bool {
	if len(value) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}

func appMain(args []string, env []string) int {
	if len(args) < 1 {
		print("missing args\n")
		return 1
	}
	for i := 0; i < len(env); i++ {
		if preparedHasPrefix(env[i], "PATH=") {
			print("PASS\n")
			return 0
		}
	}
	print("missing env\n")
	return 1
}
