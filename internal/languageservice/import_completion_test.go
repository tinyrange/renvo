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
	call := `package main
import "os"
func main() { os.WriteFile("hello.txt", []byte("Hello, World"), os.`
	if got := ImportPathAt([]byte(call), len(call)); got.Ok {
		t.Fatalf("string arguments made selector look like an import: %#v", got)
	}
}

func TestParseImportsUsesFrontendDeclarationsDespiteLaterSyntaxError(t *testing.T) {
	source := []byte(`package main
import (
	host "os"
	_ "embed"
)
func main() { host.WriteFile("hello.txt", nil, host.
`)
	got := ParseImports(source)
	if len(got) != 2 || got[0].Name != "host" || got[0].Path != "os" || got[1].Name != "_" || got[1].Path != "embed" {
		t.Fatalf("parsed imports = %#v", got)
	}
}

func TestSelectorAtUsesFrontendTokensAfterStringArguments(t *testing.T) {
	source := []byte(`package main
import "os"
func main() { os.WriteFile("hello.txt", []byte("Hello, World"), os.`)
	got := SelectorAt(source, len(source))
	if !got.Ok || got.Base != "os" || got.Prefix != "" || got.ReplaceStart != len(source) {
		t.Fatalf("selector context = %#v", got)
	}
	stringSource := []byte(`package main
var value = "os.`)
	if got := SelectorAt(stringSource, len(stringSource)); got.Ok {
		t.Fatalf("string contents became selector context: %#v", got)
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
