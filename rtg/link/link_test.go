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
		References: []unit.Symbol{
			{ImportPath: "example.com/app/pkg/answer", Name: "Value", UnitName: "rtg_example_com_app_pkg_answer_Value"},
		},
		Decls: []unit.Decl{
			{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return rtg_example_com_app_pkg_answer_Value() }\n"},
		},
	}
	depUnit := unit.Unit{
		ImportPath: "example.com/app/pkg/answer",
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
	_, err := Build([]unit.Unit{{
		ImportPath: "example.com/app/main",
		Package:    "main",
		References: []unit.Symbol{
			{ImportPath: "example.com/app/pkg/missing", Name: "Value", UnitName: "rtg_example_com_app_pkg_missing_Value"},
		},
		Decls: []unit.Decl{
			{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return rtg_example_com_app_pkg_missing_Value() }\n"},
		},
	}})
	if err == nil {
		t.Fatalf("Build succeeded with unresolved reference")
	}
	if !strings.Contains(err.Error(), "unresolved reference") {
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
