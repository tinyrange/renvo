package link

import (
	"strings"
	"testing"

	"j5.nz/rtg/rtg/unit"
)

func TestBuildResolvesReferences(t *testing.T) {
	mainUnit := unit.Unit{
		ImportPath: "example.com/app/main",
		Package:    "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		References: []unit.Symbol{
			{ImportPath: "example.com/app/pkg/answer", Name: "Value", UnitName: "rtg_example_com_app_pkg_answer_Value"},
		},
		Decls: []unit.Decl{
			{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return rtg_example_com_app_pkg_answer_Value() }\n"},
		},
	}
	depUnit := unit.Unit{
		ImportPath: "example.com/app/pkg/answer",
		Package:    "answer",
		Exports: []unit.Symbol{
			{ImportPath: "example.com/app/pkg/answer", Name: "Value", UnitName: "rtg_example_com_app_pkg_answer_Value"},
		},
	}
	plan, err := Build([]unit.Unit{mainUnit, depUnit})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(plan.Units) != 2 || plan.Units[0].ImportPath != "example.com/app/main" {
		t.Fatalf("plan ordering = %#v", plan.Units)
	}
}

func TestBuildRejectsUnresolvedReference(t *testing.T) {
	_, err := Build([]unit.Unit{
		{
			ImportPath: "example.com/app/main",
			Package:    "main",
			Imports:    []string{"example.com/app/pkg/missing"},
			References: []unit.Symbol{
				{ImportPath: "example.com/app/pkg/missing", Name: "Value", UnitName: "rtg_example_com_app_pkg_missing_Value"},
			},
			Decls: []unit.Decl{
				{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return rtg_example_com_app_pkg_missing_Value() }\n"},
			},
		},
		{
			ImportPath: "example.com/app/pkg/missing",
			Package:    "missing",
		},
	})
	if err == nil {
		t.Fatalf("Build succeeded with unresolved reference")
	}
	if !strings.Contains(err.Error(), "unresolved reference") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsReferenceWithoutImportMetadata(t *testing.T) {
	_, err := Build([]unit.Unit{
		{
			ImportPath: "example.com/app/main",
			Package:    "main",
			References: []unit.Symbol{
				{ImportPath: "example.com/app/dep", Name: "Value", UnitName: "rtg_example_com_app_dep_Value"},
			},
			Decls: []unit.Decl{
				{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return rtg_example_com_app_dep_Value() }\n"},
			},
		},
		{
			ImportPath: "example.com/app/dep",
			Package:    "dep",
			Exports:    []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Value", UnitName: "rtg_example_com_app_dep_Value"}},
		},
	})
	if err == nil {
		t.Fatalf("Build succeeded with reference missing import metadata")
	}
	if !strings.Contains(err.Error(), "example.com/app/main: reference example.com/app/dep.Value missing import metadata") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsDuplicateUnits(t *testing.T) {
	_, err := Build([]unit.Unit{
		{
			ImportPath: "example.com/app/dep",
			Package:    "dep",
		},
		{
			ImportPath: "example.com/app/dep",
			Package:    "dep",
		},
	})
	if err == nil {
		t.Fatalf("Build succeeded with duplicate unit identity")
	}
	if !strings.Contains(err.Error(), "duplicate unit: example.com/app/dep") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsInvalidUnitMetadata(t *testing.T) {
	tests := []struct {
		name  string
		units []unit.Unit
		want  string
	}{
		{
			name: "empty import path",
			units: []unit.Unit{{
				Package: "main",
			}},
			want: "empty unit import path",
		},
		{
			name: "empty package",
			units: []unit.Unit{{
				ImportPath: "example.com/app/main",
			}},
			want: "example.com/app/main: empty unit package",
		},
		{
			name: "empty import metadata",
			units: []unit.Unit{{
				ImportPath: "example.com/app/main",
				Package:    "main",
				Imports:    []string{""},
			}},
			want: "example.com/app/main: empty import metadata",
		},
		{
			name: "duplicate import metadata",
			units: []unit.Unit{{
				ImportPath: "example.com/app/main",
				Package:    "main",
				Imports:    []string{"example.com/app/dep", "example.com/app/dep"},
			}},
			want: `example.com/app/main: duplicate import metadata "example.com/app/dep"`,
		},
		{
			name: "invalid export metadata",
			units: []unit.Unit{{
				ImportPath: "example.com/app/main",
				Package:    "main",
				Exports:    []unit.Symbol{{ImportPath: "example.com/app/main", Name: "Value"}},
			}},
			want: "example.com/app/main: invalid export metadata",
		},
		{
			name: "duplicate export metadata",
			units: []unit.Unit{{
				ImportPath: "example.com/app/main",
				Package:    "main",
				Exports: []unit.Symbol{
					{ImportPath: "example.com/app/main", Name: "Value", UnitName: "rtg_example_com_app_main_Value"},
					{ImportPath: "example.com/app/main", Name: "Value", UnitName: "rtg_example_com_app_main_Value"},
				},
			}},
			want: "example.com/app/main: duplicate export metadata Value",
		},
		{
			name: "invalid reference metadata",
			units: []unit.Unit{{
				ImportPath: "example.com/app/main",
				Package:    "main",
				References: []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Value"}},
			}},
			want: "example.com/app/main: invalid reference metadata",
		},
		{
			name: "duplicate reference metadata",
			units: []unit.Unit{{
				ImportPath: "example.com/app/main",
				Package:    "main",
				References: []unit.Symbol{
					{ImportPath: "example.com/app/dep", Name: "Value", UnitName: "rtg_example_com_app_dep_Value"},
					{ImportPath: "example.com/app/dep", Name: "Value", UnitName: "rtg_example_com_app_dep_Value"},
				},
			}},
			want: "example.com/app/main: duplicate reference metadata example.com/app/dep.Value",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := Build(tt.units)
			if err == nil {
				t.Fatalf("Build accepted %s metadata", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestBuildRejectsMissingImportedUnit(t *testing.T) {
	_, err := Build([]unit.Unit{{
		ImportPath: "example.com/app/main",
		Package:    "main",
		Imports:    []string{"example.com/app/dep"},
		Decls: []unit.Decl{
			{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return 0 }\n"},
		},
	}})
	if err == nil {
		t.Fatalf("Build succeeded with missing imported unit")
	}
	if !strings.Contains(err.Error(), "example.com/app/main: missing imported unit example.com/app/dep") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsExportFromDifferentUnit(t *testing.T) {
	_, err := Build([]unit.Unit{{
		ImportPath: "example.com/app/dep",
		Package:    "dep",
		Exports: []unit.Symbol{
			{ImportPath: "example.com/app/other", Name: "Value", UnitName: "rtg_example_com_app_other_Value"},
		},
	}})
	if err == nil {
		t.Fatalf("Build succeeded with export owned by another unit")
	}
	if !strings.Contains(err.Error(), "example.com/app/dep: export Value belongs to example.com/app/other") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsDeclarationSymbolMismatch(t *testing.T) {
	_, err := Build([]unit.Unit{{
		ImportPath: "example.com/app/main",
		Package:    "main",
		Decls: []unit.Decl{
			{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func wrong() int { return 0 }\n"},
		},
	}})
	if err == nil {
		t.Fatalf("Build succeeded with mismatched declaration symbol")
	}
	if !strings.Contains(err.Error(), "example.com/app/main: declaration appMain body does not define rtg_example_com_app_main_appMain") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsEmptySymbolDeclarationBody(t *testing.T) {
	_, err := Build([]unit.Unit{{
		ImportPath: "example.com/app/main",
		Package:    "main",
		Decls: []unit.Decl{
			{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain"},
		},
	}})
	if err == nil {
		t.Fatalf("Build succeeded with empty declaration body")
	}
	if !strings.Contains(err.Error(), "example.com/app/main: declaration appMain has empty body") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsMissingEntrypoint(t *testing.T) {
	_, err := Build([]unit.Unit{{
		ImportPath: "example.com/app/main",
		Package:    "main",
		Decls: []unit.Decl{
			{Kind: "func", Name: "main", UnitName: "rtg_example_com_app_main_main", Body: "func rtg_example_com_app_main_main() {}\n"},
		},
	}})
	if err == nil {
		t.Fatalf("Build succeeded without appMain")
	}
	if !strings.Contains(err.Error(), "missing entrypoint") {
		t.Fatalf("error = %q", err)
	}
}

func TestBuildRejectsMultipleEntrypoints(t *testing.T) {
	_, err := Build([]unit.Unit{
		{
			ImportPath: "example.com/app/cmd/one",
			Package:    "main",
			Decls: []unit.Decl{
				{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_cmd_one_appMain", Body: "func rtg_example_com_app_cmd_one_appMain() int { return 0 }\n"},
			},
		},
		{
			ImportPath: "example.com/app/cmd/two",
			Package:    "main",
			Decls: []unit.Decl{
				{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_cmd_two_appMain", Body: "func rtg_example_com_app_cmd_two_appMain() int { return 0 }\n"},
			},
		},
	})
	if err == nil {
		t.Fatalf("Build succeeded with multiple appMain entrypoints")
	}
	if !strings.Contains(err.Error(), "multiple entrypoints") {
		t.Fatalf("error = %q", err)
	}
}

func TestSourceCombinesUnitsAndAddsAppMainWrapper(t *testing.T) {
	plan, err := Build([]unit.Unit{
		{
			ImportPath: "example.com/app/main",
			Package:    "main",
			Imports:    []string{"example.com/app/dep"},
			Decls: []unit.Decl{
				{
					Kind:     "func",
					Name:     "appMain",
					UnitName: "rtg_example_com_app_main_appMain",
					Body:     "func rtg_example_com_app_main_appMain(args []string) int { return rtg_example_com_app_dep_Value() }\n",
				},
			},
			References: []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Value", UnitName: "rtg_example_com_app_dep_Value"}},
		},
		{
			ImportPath: "example.com/app/dep",
			Package:    "dep",
			Exports:    []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Value", UnitName: "rtg_example_com_app_dep_Value"}},
			Decls:      []unit.Decl{{Kind: "func", Name: "Value", UnitName: "rtg_example_com_app_dep_Value", Body: "func rtg_example_com_app_dep_Value() int { return 7 }\n"}},
		},
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	src := string(Source(plan))
	for _, want := range []string{
		"package main\n",
		"func rtg_example_com_app_dep_Value() int { return 7 }\n",
		"func rtg_example_com_app_main_appMain(args []string) int",
		"func appMain(args []string) int {\n\treturn rtg_example_com_app_main_appMain(args)\n}\n",
	} {
		if !strings.Contains(src, want) {
			t.Fatalf("linked source missing %q:\n%s", want, src)
		}
	}
}
