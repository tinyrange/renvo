package main

import (
	"bytes"
	"encoding/json"
	"io"
)

func checkDecode() {
	source := []byte(`{"version":1,"root":{"kind":"int","n":9223372036854775807},"values":[{"kind":"string","text":"round-trip"}],"fields":{"a":{"kind":"bool"}},"pointer":{"kind":"none"},"quoted":"42","quotedText":"\"quoted\""}`)
	var graph Graph
	if err := json.Unmarshal(source, &graph); err != nil {
		panic(err.Error())
	}
	if graph.Version != 1 || graph.Root.N != 9223372036854775807 || graph.Root.Kind != "int" {
		panic("decoded root")
	}
	if len(graph.Values) != 1 || graph.Values[0].Text != "round-trip" || graph.Fields["a"].Kind != "bool" {
		panic("decoded collections")
	}
	if graph.Pointer == nil || graph.Pointer.Kind != "none" || graph.Quoted != 42 || graph.QuotedText != "quoted" {
		panic("decoded pointer/options")
	}
	encoded, err := json.Marshal(graph)
	if err != nil {
		panic(err.Error())
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var restored Graph
	if err := decoder.Decode(&restored); err != nil {
		panic(err.Error())
	}
	if restored.Root.N != graph.Root.N || restored.Values[0].Text != graph.Values[0].Text {
		panic("round trip")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		panic("trailing EOF")
	}
	decoder = json.NewDecoder(bytes.NewReader([]byte(`{"unknown":true}`)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&restored); err == nil {
		panic("unknown accepted")
	}
}
