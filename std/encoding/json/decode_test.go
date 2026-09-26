package json

import "testing"

//renvo:reflect
type decodeChild struct {
	Name string `json:"name"`
	N    int    `json:"n"`
}

//renvo:reflect
type decodeRecord struct {
	Child   *decodeChild           `json:"child"`
	Rows    []decodeChild          `json:"rows"`
	Lookup  map[string]decodeChild `json:"lookup"`
	Fixed   [2]int                 `json:"fixed"`
	Bytes   []byte                 `json:"bytes"`
	Label   scalarTestLabel        `json:"label"`
	Count   uint64                 `json:"count,string"`
	Dynamic any                    `json:"dynamic"`
	hidden  int
}

func TestUnmarshalRecursive(t *testing.T) {
	data := []byte(`{"child":{"name":"nested","n":7},"rows":[{"name":"first"},{"n":9}],"lookup":{"a":{"name":"mapped"}},"fixed":[3],"bytes":"AP8=","label":"named","count":"18446744073709551615","dynamic":{"values":[null,true,2.5]}}`)
	var value decodeRecord
	value.hidden = 42
	if err := Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	if value.Child == nil || value.Child.Name != "nested" || value.Child.N != 7 {
		t.Fatal("pointer")
	}
	if len(value.Rows) != 2 || value.Rows[0].Name != "first" || value.Rows[1].N != 9 {
		t.Fatal("slice")
	}
	if value.Lookup["a"].Name != "mapped" || value.Fixed[0] != 3 || value.Fixed[1] != 0 {
		t.Fatal("map/array")
	}
	if len(value.Bytes) != 2 || value.Bytes[1] != 255 || value.Label != "named" || value.Count != 18446744073709551615 {
		t.Fatal("scalars/bytes")
	}
	if value.hidden != 42 {
		t.Fatal("private field changed")
	}
	object, ok := value.Dynamic.(map[string]any)
	if !ok {
		t.Fatal("dynamic object")
	}
	items, ok := object["values"].([]any)
	if !ok || len(items) != 3 || items[0] != nil || items[1] != true || items[2] != float64(2.5) {
		t.Fatal("dynamic values")
	}
}

func TestUnmarshalReuseAndNull(t *testing.T) {
	value := decodeRecord{Child: &decodeChild{Name: "keep", N: 2}, Lookup: map[string]decodeChild{"keep": {Name: "old"}}, Rows: []decodeChild{{Name: "reuse", N: 1}}, Fixed: [2]int{7, 8}, Dynamic: "old"}
	originalChild := value.Child
	if err := Unmarshal([]byte(`{"child":{"n":3},"rows":[{"n":4}],"lookup":{"new":{"n":5}},"fixed":[],"dynamic":null}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.Child.Name != "keep" || value.Child.N != 3 || value.Rows[0].Name != "reuse" || value.Rows[0].N != 4 {
		t.Fatal("reuse")
	}
	if value.Child != originalChild || originalChild.N != 3 {
		t.Fatal("pointer identity")
	}
	if len(value.Lookup) != 2 || value.Lookup["keep"].Name != "old" || value.Lookup["new"].N != 5 {
		t.Fatal("map merge")
	}
	if value.Fixed[0] != 0 || value.Fixed[1] != 0 || value.Dynamic != nil {
		t.Fatal("zeroing")
	}
	if err := Unmarshal([]byte(`{"child":null,"rows":[],"lookup":null,"bytes":null}`), &value); err != nil {
		t.Fatal(err)
	}
	if value.Child != nil || value.Rows == nil || len(value.Rows) != 0 || value.Lookup != nil || value.Bytes != nil {
		t.Fatal("nil versus empty")
	}
}

func TestUnmarshalSyntaxAndStrictFields(t *testing.T) {
	value := decodeChild{Name: "unchanged"}
	if err := Unmarshal([]byte(`{"name":"changed"} trailing`), &value); err == nil || value.Name != "unchanged" {
		t.Fatal("trailing syntax")
	}
	node, _, err := parseJSON([]byte(`{"unknown":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := decodeTarget(node, &value, true); err == nil {
		t.Fatal("unknown field accepted")
	}
	if err := decodeTarget(node, &value, false); err != nil {
		t.Fatal(err)
	}
	var absent *decodeChild
	if Unmarshal([]byte("{}"), absent) == nil || Unmarshal([]byte("{}"), value) == nil {
		t.Fatal("invalid target accepted")
	}
	var dynamic any = "old"
	if err := Unmarshal([]byte("[null,1]"), &dynamic); err != nil {
		t.Fatal(err)
	}
	list, ok := dynamic.([]any)
	if !ok || len(list) != 2 || list[0] != nil || list[1] != float64(1) {
		t.Fatal("dynamic target")
	}
}
