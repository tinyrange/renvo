package main

func appMain(args []string) int {
	i := 0
	for i < 3 {
		fd := open("renvo_0750_loop.tmp", O_RDWR|O_CREATE|O_TRUNC)
		if fd < 0 {
			print("RENVO-0750 loop open failed\n")
			return 1
		}
		if close(fd) != 0 {
			print("RENVO-0750 loop close failed\n")
			return 2
		}
		i = i + 1
	}
	print("PASS\n")
	return 0
}
