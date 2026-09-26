//go:build !renvo

package json

import (
	stdjson "encoding/json"
	"reflect"
	"testing"
)

func TestUnmarshalAgainstGo(t *testing.T) {
	cases := []string{
		`{}`,
		`null`,
		`{"child":{"name":"keep"},"child":{"n":7}}`,
		`{"rows":[{"name":"a"},{"n":3}],"lookup":{"x":{"n":1},"x":{"name":"last"}}}`,
		`{"fixed":[1,2,3,4],"bytes":"AAEC/w==","dynamic":[null,{},[],true,false,1.25]}`,
		`{"count":"18446744073709551615","label":"\ud83d\ude00"}`,
		`{"child":null,"rows":[],"lookup":{},"bytes":null,"dynamic":null}`,
	}
	for _, source := range cases {
		got := decodeRecord{Child: &decodeChild{Name: "initial"}, Lookup: map[string]decodeChild{"old": {N: 9}}, hidden: 42}
		want := decodeRecord{Child: &decodeChild{Name: "initial"}, Lookup: map[string]decodeChild{"old": {N: 9}}, hidden: 42}
		err := Unmarshal([]byte(source), &got)
		wantErr := stdjson.Unmarshal([]byte(source), &want)
		if (err == nil) != (wantErr == nil) || !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got %#v (%v), want %#v (%v)", source, got, err, want, wantErr)
		}
	}
}
