package sha256

import (
	"encoding/hex"
	"testing"
)

func TestSum256Vectors(t *testing.T) {
	vectors := []struct {
		input string
		want  string
	}{
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"abcdefghbcdefghicdefghijdefghijkefghijklfghijklmghijklmn" +
			"hijklmnoijklmnopjklmnopqklmnopqrlmnopqrsmnopqrstnopqrstu",
			"cf5b16a778af8380036ce59e7b0492370b249b11e8f07a51afac45037afee9d1"},
	}
	for _, vector := range vectors {
		sum := Sum256([]byte(vector.input))
		if got := hex.EncodeToString(sum[:]); got != vector.want {
			t.Fatalf("Sum256(%q) = %s, want %s", vector.input, got, vector.want)
		}
	}
}
