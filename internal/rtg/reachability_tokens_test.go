package rtg

import (
	"reflect"
	"testing"
)

func TestEmbeddedReachabilityTokenRanges(t *testing.T) {
	document := Parse([]byte(`definition 1
unit reachability
implements direct_emitter_v1
arch sample {}
 go backend {
 type payload struct { value int }
 var state payload
 const text = "unused"
 func root() int { /* unused() */ return later()+len(text) }
 func unused() int { return 99 }
 func later() int { if state.value == 0 { return root() }; return state.value }
 }
 `), "reachability.rtg")
	if !document.Ok {
		t.Fatalf("parse: %#v", document.Diagnostics)
	}
	for _, tc := range []struct {
		excluded []string
		want     []string
	}{
		{nil, []string{"payload", "state", "text", "root", "later"}},
		{[]string{"later"}, []string{"text", "root"}},
	} {
		parts := reachableEmbeddedGoParts(document, []string{"root"}, tc.excluded)
		var names []string
		for _, part := range parts {
			names = append(names, part.name)
		}
		if !reflect.DeepEqual(names, tc.want) {
			t.Fatalf("excluded=%v: got %v, want %v", tc.excluded, names, tc.want)
		}
	}
}
