package main

func appMain(args []string) int {
	bs := renvo0555Make('q')
	if len(bs) != 1 || bs[0] != 'q' {
		print("RENVO-0555 append byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func renvo0555Make(b byte) []byte {
	var bs []byte
	bs = append(bs, b)
	return bs
}
