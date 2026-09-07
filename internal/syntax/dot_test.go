package syntax

import "testing"

func TestNumericDotBoundaries(t *testing.T) {
	for _, test := range []struct {
		source string
		want   []string
	}{
		{"1...", []string{"1.", ".", "."}},
		{"1....", []string{"1.", "..."}},
		{".1...", []string{".1", "..."}},
		{".1.2", []string{".1", ".2"}},
		{"0x1.2p3.4", []string{"0x1.2p3", ".4"}},
		{"0x1.2p3...", []string{"0x1.2p3", "..."}},
		{"1e2...", []string{"1e2", "..."}},
		{"1i...", []string{"1i", "..."}},
		{"1 ...", []string{"1", "..."}},
	} {
		tokens := Scan([]byte(test.source))
		if len(tokens) != len(test.want)+1 {
			t.Fatalf("%q: token count %d", test.source, len(tokens))
		}
		for i, want := range test.want {
			if got := string(TokenText([]byte(test.source), tokens[i])); got != want {
				t.Errorf("%q token %d: %q, want %q", test.source, i, got, want)
			}
		}
	}
}

func TestMalformedDotSyntax(t *testing.T) {
	for _, source := range []string{
		"func main(){_=append([]int{},1...)}",
		"var x = f(1...)",
		"func main(){_=x..Field}",
		"func main(){_=x. ...Field}",
		"func main(){_=func(){_=x..Field}}",
	} {
		file := ParseFile([]byte("package main\n" + source))
		if file.Ok || file.Error != ParseErrDot {
			t.Fatalf("accepted malformed dots in %q: %+v", source, file)
		}
	}
	for _, source := range []string{
		"func main(){_=x.}", // retain declarations for incomplete editor buffers
		"import . \"fmt\"; func main(){Println(1)}",
		"import (. \"fmt\"); func main(){Println(1)}",
		"func main(){_=x.Field;_=x.(int);switch x.(type){}}",
		"func f(x ...int){};func main(){f([]int{1}...);_= [...]int{1};_= .5+1.}",
	} {
		if file := ParseFile([]byte("package main\n" + source)); !file.Ok {
			t.Fatalf("rejected valid dots in %q: %+v", source, file)
		}
	}
}
