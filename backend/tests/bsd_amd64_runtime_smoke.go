package main

var startupValue = preserveStartupStack(7)

func preserveStartupStack(value int) int { return value }

func appMain(args []string, env []string) int {
	if startupValue != 7 || len(args) == 0 || len(env) == 0 {
		print("RENVO-BSD argv or environment missing\n")
		return 1
	}
	fd := open("renvo_bsd_runtime.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-BSD open failed\n")
		return 1
	}
	seed := []byte("bsd")
	if write(fd, seed, 0) != 3 || chmod(fd, 420) != 0 {
		print("RENVO-BSD write or chmod failed\n")
		return 1
	}
	got := []byte("   ")
	if read(fd, got, 0) != 3 || string(got) != "bsd" || close(fd) != 0 {
		print("RENVO-BSD read or close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
