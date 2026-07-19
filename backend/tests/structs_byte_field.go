package main

type Renvo0604Byte struct{ b byte }

func renvo0604Make(b byte) Renvo0604Byte {
	return Renvo0604Byte{b: b}
}

func appMain(args []string) int {
	if renvo0604Make('m').b != 'm' {
		print("RENVO-0604 byte field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
