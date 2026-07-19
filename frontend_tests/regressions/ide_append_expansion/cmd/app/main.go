package main

func main() {
	buffer := []byte{10, 20, 30, 40, 50}
	out := []byte{1, 2}
	out = append(out, buffer[1:4]...)
	if len(out) != 5 || out[0] != 1 || out[1] != 2 || out[2] != 20 || out[3] != 30 || out[4] != 40 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
	return
}
