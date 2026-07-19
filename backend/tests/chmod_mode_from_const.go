package main

const renvo0781Mode = 420

func appMain(args []string) int {
	fd := open("renvo_0781_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0781 open failed\n")
		return 1
	}
	if chmod(fd, renvo0781Mode) != 0 {
		print("RENVO-0781 chmod failed\n")
		return 1
	}
	if close(fd) != 0 {
		print("RENVO-0781 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
