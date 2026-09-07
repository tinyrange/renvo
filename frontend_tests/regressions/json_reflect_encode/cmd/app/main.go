package main

import (
	"encoding/json"
	"math"
)

type Kind string
type Octet uint8

//renvo:reflect
type Value struct {
	Kind Kind   `json:"kind"`
	N    int64  `json:"n,omitempty"`
	Text string `json:"text,omitempty"`
}

//renvo:reflect
type Graph struct {
	Version    int              `json:"version"`
	Root       Value            `json:"root"`
	Values     []Value          `json:"values"`
	Fields     map[string]Value `json:"fields"`
	Bytes      []Octet          `json:"bytes"`
	Empty      []int            `json:"empty"`
	Absent     []int            `json:"absent"`
	Pointer    *Value           `json:"pointer,omitempty"`
	Quoted     int              `json:"quoted,string"`
	QuotedText string           `json:"quotedText,string"`
	Skip       string           `json:"-"`
	hidden     string
}

func check(value any, want string) {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err.Error() + " expected " + want)
	}
	if string(data) != want {
		panic("JSON mismatch: " + string(data) + " expected " + want)
	}
}

func main() {
	checkDecode()
	checkRenvoPolicy()
	for _, source := range []string{`null`, `{"n":18446744073709551615,"a":[true,false]}`, `"\ud800\u0061"`, `1e99999999999999999`, "\"\xff\""} {
		if !json.Valid([]byte(source)) {
			panic("rejected JSON syntax")
		}
	}
	for _, source := range []string{"01", "[1,]", "true false", "\"\\x00\"", "{\"a\":}", "1e+"} {
		if json.Valid([]byte(source)) {
			panic("accepted invalid JSON syntax")
		}
	}
	g := Graph{Version: 2, Root: Value{Kind: "int", N: 42}, Values: []Value{{Kind: "str", Text: "<>&\n雪\u2028"}}, Fields: map[string]Value{"z": {Kind: "nil"}, "a": {Kind: "bool", N: 1}}, Bytes: []Octet{0, 255}, Empty: []int{}, Pointer: &Value{Kind: "int", N: -7}, Quoted: 15, QuotedText: "quoted\n", Skip: "not serialized", hidden: "private"}
	check(g, `{"version":2,"root":{"kind":"int","n":42},"values":[{"kind":"str","text":"\u003c\u003e\u0026\n雪\u2028"}],"fields":{"a":{"kind":"bool","n":1},"z":{"kind":"nil"}},"bytes":"AP8=","empty":[],"absent":null,"pointer":{"kind":"int","n":-7},"quoted":"15","quotedText":"\"quoted\\n\""}`)
	check("bad\xffutf8", `"bad\ufffdutf8"`)
	check(uint64(18446744073709551615), "18446744073709551615")
	check(int64(-9223372036854775808), "-9223372036854775808")
	check(1e-7, "1e-7")
	check(1e-6, "0.000001")
	check(1e20, "100000000000000000000")
	check(1e21, "1e+21")
	check(float32(1.25), "1.25")
	check(float32(1e-6), "0.000001")
	check(float32(1e21), "1e+21")
	check(math.Copysign(0, -1), "-0")
	check([]byte{}, `""`)
	var data []byte
	check(data, "null")
	_, err := json.Marshal(math.Inf(1))
	if err == nil {
		panic("accepted infinity")
	}
	print("PASS\n")
}
