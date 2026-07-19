package main

func appMain(args []string) int {
	var b byte = 'r'
	p := renvo0630Ptr(&b)
	if *p != 'r' {
		print("RENVO-0630 byte address failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func renvo0630Ptr(p *byte) *byte {
	return p
}
