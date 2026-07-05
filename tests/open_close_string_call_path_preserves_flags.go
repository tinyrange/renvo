package main

func rtgOpenCString(s string) string {
	var out []byte
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return string(out)
}

func createWithCString(path string) int {
	return open(rtgOpenCString(path), O_RDWR|O_CREATE|O_TRUNC)
}

func appMain(args []string, env []string) int {
	fd := createWithCString("rtg_1003_open.tmp")
	if fd < 0 {
		print("FAIL\n")
		return 1
	}
	close(fd)
	print("PASS\n")
	return 0
}
