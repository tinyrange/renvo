package languageservice

import (
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/internal/driver"
)

func TestImportPathAtDirectGroupedAndClosedImports(t *testing.T) {
	for _, test := range []struct {
		source string
		caret  int
		prefix string
		closed bool
	}{
		{source: `package main
import "fmt`, caret: len(`package main
import "fmt`), prefix: "fmt"},
		{source: `package main
import alias "encoding/bi`, caret: len(`package main
import alias "encoding/bi`), prefix: "encoding/bi"},
		{source: "package main\nimport (\n  \"str", caret: len("package main\nimport (\n  \"str"), prefix: "str"},
		{source: `package main
import "fmt"`, caret: len(`package main
import "fmt`), prefix: "fmt", closed: true},
	} {
		got := ImportPathAt([]byte(test.source), test.caret)
		if !got.Ok || got.Prefix != test.prefix || got.Closed != test.closed {
			t.Fatalf("ImportPathAt(%q, %d) = %#v", test.source, test.caret, got)
		}
	}
	if got := ImportPathAt([]byte(`package main
func main() { println("fmt`), len(`package main
func main() { println("fmt`)); got.Ok {
		t.Fatalf("ordinary string was treated as an import: %#v", got)
	}
}

func TestCompleteStandardImportPathsUsesBundledSourceLayout(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	stdRoot := filepath.Join(filepath.Clean(filepath.Join(wd, "..", "..")), "std")
	got := CompleteStandardImportPaths(stdRoot, "linux/amd64", nil, "enc", driver.OSFS{})
	if !containsImportPath(got, "encoding/binary") {
		t.Fatalf("encoding completion = %#v", got)
	}
	all := CompleteStandardImportPaths(stdRoot, "linux/amd64", nil, "", driver.OSFS{})
	for _, want := range []string{"fmt", "strings", "unicode/utf8"} {
		if !containsImportPath(all, want) {
			t.Fatalf("standard import completion missing %q: %#v", want, all)
		}
	}
}

func containsImportPath(paths []string, want string) bool {
	for i := 0; i < len(paths); i++ {
		if paths[i] == want {
			return true
		}
	}
	return false
}
