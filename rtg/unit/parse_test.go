package unit

import (
	"strings"
	"testing"
)

func TestParseSourceRoundTripMetadata(t *testing.T) {
	u := Unit{
		ImportPath: "example.com/app/main",
		Package:    "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Exports:    []Symbol{{ImportPath: "example.com/app/main", Name: "Main", UnitName: "rtg_example_com_app_main_Main"}},
		References: []Symbol{{ImportPath: "example.com/app/pkg/answer", Name: "Value", UnitName: "rtg_example_com_app_pkg_answer_Value"}},
		Decls:      []Decl{{Kind: "func", Name: "Main", UnitName: "rtg_example_com_app_main_Main", Path: "main.go", Body: "func rtg_example_com_app_main_Main() int { return 0 }\n"}},
	}
	src := sourceForTest(u)
	parsed, err := ParseSource("main.rtg.go", src)
	if err != nil {
		t.Fatalf("ParseSource failed: %v", err)
	}
	if parsed.ImportPath != u.ImportPath || parsed.Package != u.Package {
		t.Fatalf("parsed identity = %#v", parsed)
	}
	if len(parsed.Imports) != 1 || parsed.Imports[0] != "example.com/app/pkg/answer" {
		t.Fatalf("imports = %#v", parsed.Imports)
	}
	if len(parsed.Exports) != 1 || parsed.Exports[0].UnitName != "rtg_example_com_app_main_Main" {
		t.Fatalf("exports = %#v", parsed.Exports)
	}
	if len(parsed.References) != 1 || parsed.References[0].ImportPath != "example.com/app/pkg/answer" {
		t.Fatalf("references = %#v", parsed.References)
	}
	if len(parsed.Decls) != 1 || parsed.Decls[0].Name != "Main" {
		t.Fatalf("decls = %#v", parsed.Decls)
	}
	if !strings.Contains(parsed.Decls[0].Body, "func rtg_example_com_app_main_Main() int") {
		t.Fatalf("decl body was not preserved: %q", parsed.Decls[0].Body)
	}
}

func TestParseSourcesReadsLoadedUnitFiles(t *testing.T) {
	units, err := ParseSources([]SourceFile{
		{Path: "main.rtg.go", Source: sourceForTest(Unit{
			ImportPath: "example.com/app/main",
			Package:    "main",
			Decls:      []Decl{{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Path: "main.go", Body: "func rtg_example_com_app_main_appMain() int { return 0 }\n"}},
		})},
		{Path: "dep.rtg.go", Source: sourceForTest(Unit{
			ImportPath: "example.com/app/dep",
			Package:    "dep",
		})},
	})
	if err != nil {
		t.Fatalf("ParseSources failed: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("units = %#v, want 2", units)
	}
	if units[0].ImportPath != "example.com/app/main" || units[1].ImportPath != "example.com/app/dep" {
		t.Fatalf("unit order/identity = %#v", units)
	}
}

func TestParseSourcesReportsSourcePath(t *testing.T) {
	_, err := ParseSources([]SourceFile{{Path: "broken.rtg.go", Source: []byte("package main\n")}})
	if err == nil {
		t.Fatalf("ParseSources accepted source without unit metadata")
	}
	if !strings.Contains(err.Error(), "broken.rtg.go: missing rtg unit metadata") {
		t.Fatalf("error = %q", err)
	}
}

func TestParseSourceRejectsUnknownMetadata(t *testing.T) {
	_, err := ParseSource("unit.rtg.go", []byte(`//go:build rtg

// rtg:unit example.com/app
package main

// rtg:unknown value
`))
	if err == nil {
		t.Fatalf("ParseSource accepted unknown rtg metadata")
	}
	if !strings.Contains(err.Error(), `unit.rtg.go: unknown rtg metadata "unknown value"`) {
		t.Fatalf("error = %q", err)
	}
}

func TestParseSourceRequiresUnitMetadataFirst(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "import",
			src: `//go:build rtg

// rtg:import "example.com/app/dep"
// rtg:unit example.com/app
package main
`,
		},
		{
			name: "decl",
			src: `//go:build rtg

// rtg:decl func appMain => rtg_example_com_app_appMain main.go
func rtg_example_com_app_appMain() int { return 0 }
// rtg:unit example.com/app
package main
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSource(tt.name+".rtg.go", []byte(tt.src))
			if err == nil {
				t.Fatalf("ParseSource accepted %s metadata before unit identity", tt.name)
			}
			if !strings.Contains(err.Error(), "rtg metadata before unit declaration") {
				t.Fatalf("error = %q", err)
			}
		})
	}
}

func TestParseSourceRejectsDuplicateMetadata(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "unit",
			src: `// rtg:unit example.com/app
// rtg:unit example.com/app
package main
`,
			want: "duplicate rtg unit metadata",
		},
		{
			name: "import",
			src: `// rtg:unit example.com/app
package main
// rtg:import "example.com/app/dep"
// rtg:import "example.com/app/dep"
`,
			want: `duplicate import metadata "example.com/app/dep"`,
		},
		{
			name: "export",
			src: `// rtg:unit example.com/app
package main
// rtg:export Value => rtg_example_com_app_Value
// rtg:export Value => rtg_example_com_app_Value
`,
			want: "duplicate export metadata Value",
		},
		{
			name: "reference",
			src: `// rtg:unit example.com/app
package main
// rtg:ref example.com/app/dep Value => rtg_example_com_app_dep_Value
// rtg:ref example.com/app/dep Value => rtg_example_com_app_dep_Value
`,
			want: "duplicate reference metadata example.com/app/dep.Value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSource(tt.name+".rtg.go", []byte(tt.src))
			if err == nil {
				t.Fatalf("ParseSource accepted duplicate %s metadata", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestParseSourceRejectsDeclarationMetadataWithoutBody(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "before metadata",
			src: `// rtg:unit example.com/app
package main
// rtg:decl func appMain => rtg_example_com_app_appMain main.go
// rtg:import "example.com/app/dep"
`,
			want: "declaration metadata for appMain has no body before next rtg metadata",
		},
		{
			name: "at eof",
			src: `// rtg:unit example.com/app
package main
// rtg:decl func appMain => rtg_example_com_app_appMain main.go
`,
			want: "declaration metadata for appMain has no body",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSource(tt.name+".rtg.go", []byte(tt.src))
			if err == nil {
				t.Fatalf("ParseSource accepted declaration metadata without body")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestParseSourceRejectsEmptyMetadataPaths(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "unit",
			src:  "// rtg:unit " + "\npackage main\n",
			want: "empty rtg unit metadata",
		},
		{
			name: "import",
			src: `// rtg:unit example.com/app
package main
// rtg:import ""
`,
			want: "empty import metadata",
		},
		{
			name: "reference",
			src: `// rtg:unit example.com/app
package main
// rtg:ref  Value => rtg_value
`,
			want: `invalid symbol metadata "=> rtg_value"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSource(tt.name+".rtg.go", []byte(tt.src))
			if err == nil {
				t.Fatalf("ParseSource accepted empty %s metadata", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

func sourceForTest(u Unit) []byte {
	var b strings.Builder
	b.WriteString("//go:build rtg\n\n")
	b.WriteString("// Code generated by rtg; DO NOT EDIT.\n")
	b.WriteString("// rtg:unit ")
	b.WriteString(u.ImportPath)
	b.WriteString("\npackage ")
	b.WriteString(u.Package)
	b.WriteString("\n\n")
	for _, imp := range u.Imports {
		b.WriteString("// rtg:import \"")
		b.WriteString(imp)
		b.WriteString("\"\n")
	}
	for _, sym := range u.Exports {
		b.WriteString("// rtg:export ")
		b.WriteString(sym.Name)
		b.WriteString(" => ")
		b.WriteString(sym.UnitName)
		b.WriteByte('\n')
	}
	for _, sym := range u.References {
		b.WriteString("// rtg:ref ")
		b.WriteString(sym.ImportPath)
		b.WriteByte(' ')
		b.WriteString(sym.Name)
		b.WriteString(" => ")
		b.WriteString(sym.UnitName)
		b.WriteByte('\n')
	}
	for _, decl := range u.Decls {
		b.WriteString("// rtg:decl ")
		b.WriteString(decl.Kind)
		b.WriteByte(' ')
		b.WriteString(decl.Name)
		b.WriteString(" => ")
		b.WriteString(decl.UnitName)
		b.WriteByte(' ')
		b.WriteString(decl.Path)
		b.WriteByte('\n')
		b.WriteString(decl.Body)
	}
	return []byte(b.String())
}
