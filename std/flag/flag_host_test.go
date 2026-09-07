//go:build !renvo

package flag

import (
	"bytes"
	standard "flag"
	"reflect"
	"testing"
)

func TestFlagsAgainstGo(t *testing.T) {
	for _, args := range [][]string{{}, {"-gas=0x10", "-verbose=false", "file"}, {"--text=", "-verbose", "--", "-gas"}, {"-text", "false", "-gas", "010", "file"}, {"-h"}, {"-unknown"}, {"-gas"}} {
		got := NewFlagSet("test", ContinueOnError)
		want := standard.NewFlagSet("test", standard.ContinueOnError)
		var gotOut, wantOut bytes.Buffer
		got.SetOutput(&gotOut)
		want.SetOutput(&wantOut)
		g := got.Uint64("gas", 100, "budget")
		w := want.Uint64("gas", 100, "budget")
		gb := got.Bool("verbose", false, "verbose")
		wb := want.Bool("verbose", false, "verbose")
		gs := got.String("text", "false", "text")
		ws := want.String("text", "false", "text")
		ge, we := got.Parse(args), want.Parse(args)
		if (ge == nil) != (we == nil) || *g != *w || *gb != *wb || *gs != *ws || got.NFlag() != want.NFlag() || !reflect.DeepEqual(got.Args(), want.Args()) || gotOut.String() != wantOut.String() {
			t.Fatalf("%v: got %v %q, want %v %q", args, ge, gotOut.String(), we, wantOut.String())
		}
	}
}
