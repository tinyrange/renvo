package main

func appMain(args []string) int {
	var buf []byte
	buf = append(buf, 'a')
	buf = append(buf, 'b')
	buf = append(buf, 'c')
	length := int(len(buf))
	if length != 3 {
		print("RENVO-0843 len after append failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
