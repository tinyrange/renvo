package main

func appMain(args []string) int {
	fd := open("renvo_1001_open.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-1001 create failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-1001 first close failed\n")
		return 2
	}
	fd = open("renvo_1001_open.tmp", O_RDONLY)
	if fd < 0 {
		print("RENVO-1001 read-only open failed\n")
		return 3
	}
	if close(fd) != 0 {
		print("RENVO-1001 second close failed\n")
		return 4
	}
	print("PASS\n")
	return 0
}
