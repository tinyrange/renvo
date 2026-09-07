package main

import "encoding/json"

type Kind uint8

//renvo:reflect
type Value struct {
	Kind Kind   `json:"kind"`
	N    int64  `json:"n,omitempty"`
	S    string `json:"s,omitempty"`
	T    string `json:"t,omitempty"`
	Ref  int    `json:"ref,omitempty"`
}

//renvo:reflect
type Graph struct {
	Version int     `json:"version"`
	Root    Value   `json:"root"`
	Values  []Value `json:"values"`
}

func main() {
	g := Graph{Version: 1, Root: Value{Kind: 3, Ref: 1}, Values: []Value{{Kind: 2, N: 41}, {Kind: 4, S: "sample"}}}
	data, err := json.Marshal(g)
	if err != nil {
		panic(err)
	}
	if string(data) != `{"version":1,"root":{"kind":3,"ref":1},"values":[{"kind":2,"n":41},{"kind":4,"s":"sample"}]}` {
		panic(string(data))
	}
	println("PASS")
}
