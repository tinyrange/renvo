package json

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

type testEmbedded struct {
	ID int `json:"id"`
}

type testTagged struct {
	testEmbedded
	Name    string `json:"display"`
	Empty   string `json:"empty,omitempty"`
	Count   int    `json:"count,string"`
	Values  []int  `json:"values"`
	Ignored string `json:"-"`
	hidden  string
}

func TestValidAndRawMessage(t *testing.T) {
	good := []string{`null`, `true`, `-12.5e+2`, `{"a":[1,"x",false],"u":"\u20ac"}`}
	for _, s := range good {
		if !Valid([]byte(s)) {
			t.Fatalf("rejected %s", s)
		}
	}
	bad := []string{``, `01`, `{"a":}`, `[1,]`}
	for _, s := range bad {
		if Valid([]byte(s)) {
			t.Fatalf("accepted %s", s)
		}
	}
	encoded, err := Marshal(map[string]any{"z": []any{1, true, nil}, "a": "x"})
	if err != nil || string(encoded) != `{"a":"x","z":[1,true,null]}` {
		t.Fatal(string(encoded), err)
	}
	pretty, err := MarshalIndent(map[string]any{"a": []any{1, "x\n"}}, "", "  ")
	if err != nil || string(pretty) != "{\n  \"a\": [\n    1,\n    \"x\\n\"\n  ]\n}" {
		t.Fatal(string(pretty), err)
	}
	m := RawMessage(`{"ok":true}`)
	b, e := Marshal(m)
	if e != nil || string(b) != string(m) {
		t.Fatal(string(b), e)
	}
}

func TestMarshalStructTagsAndTypedSlice(t *testing.T) {
	value := &testTagged{
		testEmbedded: testEmbedded{ID: 7}, Name: "renvo", Count: 3,
		Values: []int{2, 4}, Ignored: "no", hidden: "no",
	}
	encoded, err := Marshal(value)
	if err != nil || string(encoded) != `{"id":7,"display":"renvo","count":"3","values":[2,4]}` {
		t.Fatalf("Marshal() = %q, %v", encoded, err)
	}
}

func TestUnmarshalDynamicAndTaggedStruct(t *testing.T) {
	var dynamic any
	if err := Unmarshal([]byte(`{"name":"renvo","values":[2,true,null],"unicode":"\u20ac"}`), &dynamic); err != nil {
		t.Fatal(err)
	}
	object := dynamic.(map[string]any)
	if object["name"] != "renvo" || object["unicode"] != "€" || len(object["values"].([]any)) != 3 {
		t.Fatalf("unexpected dynamic value: %#v", dynamic)
	}

	var value testTagged
	if err := Unmarshal([]byte(`{"id":9,"display":"decoded","count":"5","values":[3,6],"Ignored":"no","hidden":"no"}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.ID != 9 || value.Name != "decoded" || value.Count != 5 || len(value.Values) != 2 || value.Values[1] != 6 || value.Ignored != "" || value.hidden != "" {
		t.Fatalf("unexpected struct value: %#v", value)
	}
}

func TestTypedStringMapAndStream(t *testing.T) {
	encoded, err := Marshal(map[string]string{"b": "two", "a": "one"})
	if err != nil || string(encoded) != `{"a":"one","b":"two"}` {
		t.Fatalf("Marshal map = %q, %v", encoded, err)
	}
	var mapping map[string]string
	if err := Unmarshal(encoded, &mapping); err != nil || mapping["b"] != "two" {
		t.Fatalf("Unmarshal map = %#v, %v", mapping, err)
	}

	var output bytes.Buffer
	encoder := NewEncoder(&output)
	if err := encoder.Encode(map[string]string{"ok": "yes"}); err != nil || output.String() != "{\"ok\":\"yes\"}\n" {
		t.Fatalf("Encode = %q, %v", output.String(), err)
	}
	decoder := NewDecoder(strings.NewReader(`1 {"name":"renvo"}`))
	var number int
	var object map[string]string
	if err := decoder.Decode(&number); err != nil || number != 1 {
		t.Fatalf("first Decode = %d, %v", number, err)
	}
	if err := decoder.Decode(&object); err != nil || object["name"] != "renvo" {
		t.Fatalf("second Decode = %#v, %v", object, err)
	}
	if err := decoder.Decode(&object); err != io.EOF {
		t.Fatalf("final Decode = %v, want EOF", err)
	}
}

func TestNestedRawMessageAndRawMap(t *testing.T) {
	type holder struct {
		Data RawMessage `json:"data"`
	}
	var value holder
	if err := Unmarshal([]byte(`{"data":{"ok":true}}`), &value); err != nil || string(value.Data) != `{"ok":true}` {
		t.Fatalf("nested RawMessage = %q, %v", value.Data, err)
	}
	var mapping map[string]RawMessage
	if err := Unmarshal([]byte(`{"count":3,"ready":true}`), &mapping); err != nil || string(mapping["count"]) != "3" || string(mapping["ready"]) != "true" {
		t.Fatalf("RawMessage map = %#v, %v", mapping, err)
	}
}

func TestMarshalIndentEmptyNestedAndPrefix(t *testing.T) {
	value := map[string]any{"empty": []any{}, "nested": map[string]any{"ok": true}}
	got, err := MarshalIndent(value, "> ", "--")
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n> --\"empty\": [],\n> --\"nested\": {\n> ----\"ok\": true\n> --}\n> }"
	if string(got) != want {
		t.Fatalf("MarshalIndent() = %q, want %q", got, want)
	}
}
