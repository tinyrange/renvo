package reflect

import "testing"

type tagTestEmbedded struct {
	ID int `json:"id"`
}
type tagTestValue struct {
	tagTestEmbedded
	Name string `json:"display,omitempty" note:"line\nvalue"`
}

func TestStructFieldMetadataAndTags(t *testing.T) {
	typ := TypeOf(tagTestValue{})
	if typ == nil || typ.Kind() != Struct || typ.NumField() != 2 {
		t.Fatalf("type = %#v", typ)
	}
	if field := typ.Field(0); !field.Anonymous || field.Name != "tagTestEmbedded" {
		t.Fatalf("embedded field = %#v", field)
	}
	field := typ.Field(1)
	if got := field.Tag.Get("json"); got != "display,omitempty" {
		t.Fatalf("json tag = %q", got)
	}
	if got, ok := field.Tag.Lookup("note"); !ok || got != "line\nvalue" {
		t.Fatalf("note tag = %q, %v", got, ok)
	}
}
