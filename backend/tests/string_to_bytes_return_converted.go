package main

type Renvo0589Box struct {
	data []byte
}

func renvo0589Bytes() []byte {
	return []byte("box")
}

func appMain(args []string) int {
	b := Renvo0589Box{data: renvo0589Bytes()}
	if b.data[2] != 'x' {
		print("RENVO-0589 return converted failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
