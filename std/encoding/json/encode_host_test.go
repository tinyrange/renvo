//go:build !renvo

package json

import (
	stdjson "encoding/json"
	"math"
	"testing"
)

type namedKind string
type namedByte byte

//renvo:reflect
type testRecord struct {
	Kind          namedKind      `json:"kind"`
	Name          string         `json:"name"`
	Skip          string         `json:"-"`
	Zero          int            `json:"zero,omitempty"`
	Number        int            `json:"number,string"`
	Text          string         `json:"text,string"`
	Bytes         []namedByte    `json:"bytes"`
	Items         []int          `json:"items"`
	Map           map[string]int `json:"map"`
	Child         *testRecord    `json:"child,omitempty"`
	Dash          string         `json:"-,"`
	PointerNumber *int           `json:"pointerNumber,string,omitempty"`
	hidden        string
}

func TestMarshalAgainstGo(t *testing.T) {
	number := 8
	fixture := testRecord{Kind: "record", Name: "<>&\n雪\u2028", Skip: "no", Number: 7, Text: "quoted\n", Bytes: []namedByte{0, 255}, Items: []int{}, Map: map[string]int{"z": 2, "a": 1}, Child: &testRecord{Kind: "child"}}
	fixture.PointerNumber = &number
	cases := []any{nil, true, false, "bad\xffutf8", int64(-9223372036854775808), uint64(18446744073709551615), 1e-7, 1e-6, 1e20, 1e21, float32(1.25), float32(1e-6), float32(1e21), math.Copysign(0, -1), []byte{}, []byte(nil), fixture, map[int]string{-2: "a", 10: "b"}}
	for i, value := range cases {
		got, err := Marshal(value)
		want, werr := stdjson.Marshal(value)
		if err != nil || werr != nil || string(got) != string(want) {
			t.Fatalf("case %d: got %s (%v), want %s (%v)", i, got, err, want, werr)
		}
	}
}

func TestMarshalFloatSamplesAgainstGo(t *testing.T) {
	bits := uint64(1234567)
	for i := 0; i < 2000; i++ {
		bits = bits*6364136223846793005 + 1442695040888963407
		value := math.Float64frombits(bits)
		got, err := Marshal(value)
		want, werr := stdjson.Marshal(value)
		if (err == nil) != (werr == nil) || string(got) != string(want) {
			t.Fatalf("float bits %x: got %s (%v), want %s (%v)", bits, got, err, want, werr)
		}
	}
}

func TestMarshalRejectsInvalidValues(t *testing.T) {
	for _, value := range []any{math.NaN(), math.Inf(1), make(chan int), map[bool]int{true: 1}} {
		data, err := Marshal(value)
		if err == nil || data != nil {
			t.Fatal("accepted invalid value", value)
		}
	}
}
