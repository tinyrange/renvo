package main

func appMain(args []string) int {
	fd := open("renvo_0834_cleanup.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0834 open failed\n")
		return 1
	}
	goto cleanup
cleanup:
	if close(fd) != 0 {
		print("RENVO-0834 cleanup failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
