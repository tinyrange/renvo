package main

func appMain(args []string) int {
	fd := open("renvo_0730_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0730 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0730 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
