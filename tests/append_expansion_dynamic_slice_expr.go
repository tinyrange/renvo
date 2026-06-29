package main

func appMain() int {
	src := []byte{'P', 'A', 'S', 'S', '\n', 'x'}
	n := 5
	var out []byte
	out = append(out, src[:n]...)
	if len(out) != 5 {
		print("bad len\n")
		return 1
	}
	if out[0] != 'P' || out[1] != 'A' || out[2] != 'S' || out[3] != 'S' || out[4] != '\n' {
		print("bad copy\n")
		return 1
	}
	print(string(out))
	return 0
}
