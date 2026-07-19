package main

func appMain(args []string) int {
	bs := []byte("sum")
	p := &bs[0]
	total := int(*p)
	for i := 1; i < len(bs); i = i + 1 {
		total = total + int(bs[i])
	}
	if total != 341 {
		print("RENVO-0590 checksum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
