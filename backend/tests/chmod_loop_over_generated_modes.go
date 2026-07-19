package main

func appMain(args []string) int {
	fd := open("renvo_0794_chmod.tmp", O_RDWR|O_CREATE|O_TRUNC)
	if fd < 0 {
		print("RENVO-0794 open failed\n")
		return 1
	}
	i := 0
	for i < 3 {
		if chmod(fd, 384+i) != 0 {
			print("RENVO-0794 loop chmod failed\n")
			return 2
		}
		i = i + 1
	}
	if close(fd) != 0 {
		print("RENVO-0794 close failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
