package driver

import (
	"bytes"
	"testing"
)

func TestSourceConcurrencyImport(t *testing.T) {
	for _, src := range []string{
		"package main\n// go func(){}(); chan int\nfunc main(){}",
		"package main\nvar text = `go chan select`",
	} {
		out, needed := sourceConcurrencyImport([]byte(src))
		if needed || string(out) != src {
			t.Fatal("non-code caused implicit dependency")
		}
	}
	src := []byte("package main\nfunc main(){ c := make(chan int, 1); c <- 1 }")
	out, needed := sourceConcurrencyImport(src)
	if !needed || !bytes.Contains(out, []byte(`import _ "renvo.dev/x/runtime/serial"`)) || bytes.Count(out, []byte("\n")) != bytes.Count(src, []byte("\n")) {
		t.Fatal("missing implicit dependency or changed source lines")
	}
}
