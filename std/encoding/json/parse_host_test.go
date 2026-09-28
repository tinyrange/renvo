//go:build !renvo

package json

import (
	stdjson "encoding/json"
	"strings"
	"testing"
)

func TestValidMutationsAgainstGo(t *testing.T) {
	seeds := []string{
		`{"n":18446744073709551615,"s":"\ud800\u0061","a":[null,true,false,-1.25e+3]}`,
		`"\\\"/\b\f\n\r\t\u0000\ud834\udd1e"`,
		`[1e999999999999999999999999999999,{},[]]`,
	}
	for _, source := range seeds {
		for i := 0; i <= len(source); i++ {
			for _, b := range []byte{0, ' ', '"', '\\', '/', ':', ',', '[', ']', '{', '}', '0', '-', 0xff} {
				mutated := []byte(source[:i] + string([]byte{b}) + source[i:])
				if Valid(mutated) != stdjson.Valid(mutated) {
					t.Fatalf("validation differs: %q", mutated)
				}
			}
		}
	}
}

func TestJSONDepthLimitAgainstGo(t *testing.T) {
	for _, depth := range []int{9999, 10000, 10001} {
		source := []byte(strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth))
		if Valid(source) != stdjson.Valid(source) {
			t.Fatal("depth limit differs", depth)
		}
	}
}
