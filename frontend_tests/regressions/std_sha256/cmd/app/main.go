package main

import (
	"crypto/sha256"
	"encoding/hex"
)

func main() {
	inputs := []string{
		"",
		"abc",
		"abcdefghbcdefghicdefghijdefghijkefghijklfghijklmghijklmn" +
			"hijklmnoijklmnopjklmnopqklmnopqrlmnopqrsmnopqrstnopqrstu",
	}
	wants := []string{
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
		"cf5b16a778af8380036ce59e7b0492370b249b11e8f07a51afac45037afee9d1",
	}
	for i, input := range inputs {
		sum := sha256.Sum256([]byte(input))
		if hex.EncodeToString(sum[:]) != wants[i] {
			print("FAIL\n")
			return
		}
	}
	print("PASS\n")
}
