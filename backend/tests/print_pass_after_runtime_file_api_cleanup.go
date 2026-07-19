package main

func appMain(args []string) int {
	fd := open("renvo_0725_print_cleanup.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0725 open failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0725 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
