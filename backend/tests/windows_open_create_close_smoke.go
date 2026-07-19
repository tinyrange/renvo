package main

func appMain(args []string, env []string) int {
	fd := open("renvo-open-smoke.tmp", O_WRONLY|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("FAIL\n")
		return 1
	}
	if close(fd) != 0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
