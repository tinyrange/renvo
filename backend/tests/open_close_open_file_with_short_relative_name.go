package main

func appMain(args []string) int {
	fd := open("renvo_0747_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0747 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0747 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
