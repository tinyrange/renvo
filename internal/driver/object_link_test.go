package driver

import "testing"

func TestObjectLinkCommandSelection(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"renvo", "cc", "main.o", "-o", "app"}, true},
		{[]string{"renvo", "cc", "-o", "app.o", "main.c"}, false},
		{[]string{"renvo", "cc", "-c", "main.o"}, false},
		{[]string{"renvo", "cc", "main.o", "source.i"}, false},
		{[]string{"renvo", "main.o"}, false},
	} {
		if got := ObjectLinkCommandRequested(tc.args); got != tc.want {
			t.Errorf("%v: %v", tc.args, got)
		}
	}
}

func TestObjectLinkCommandErrors(t *testing.T) {
	for _, args := range [][]string{
		{"renvo", "cc", "-t", "darwin/arm64", "main.o"},
		{"renvo", "cc", "-o"},
		{"renvo", "cc", "-unknown"},
		{"renvo", "cc", "missing.o"},
	} {
		_, data, diagnostic := LinkObjectCommand(args, memorySourceFS{})
		if diagnostic == "" || data != nil {
			t.Errorf("accepted %v", args)
		}
	}
}
