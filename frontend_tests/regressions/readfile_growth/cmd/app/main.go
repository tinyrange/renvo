package main

import "os"

func main() {
	name := os.Args[0] + ".readfile-growth"
	for _, size := range []int{0, 1, 4095, 4096, 4097, 8192, 65536, 65539} {
		expected := make([]byte, size)
		for i := range expected {
			expected[i] = byte(i*31 + i/17)
		}
		if os.WriteFile(name, expected, 0600) != nil {
			panic("write")
		}
		actual, err := os.ReadFile(name)
		if err != nil || len(actual) != size {
			panic("read length")
		}
		for i := range expected {
			if actual[i] != expected[i] {
				panic("read contents")
			}
		}
		if len(actual) > 0 {
			actual[0] ^= 255
		}
		again, err := os.ReadFile(name)
		if err != nil || len(again) != size {
			panic("reread length")
		}
		for i := range expected {
			if again[i] != expected[i] {
				panic("read alias")
			}
		}
	}
	if data, err := os.ReadFile(name + ".missing"); err == nil || data != nil {
		panic("missing file")
	}
	print("PASS\n")
}
