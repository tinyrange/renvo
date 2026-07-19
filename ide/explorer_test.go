package ide

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExplorerSortsAndNavigatesVisibleRows(t *testing.T) {
	root := t.TempDir()
	mustMkdir(t, filepath.Join(root, "zeta"))
	mustMkdir(t, filepath.Join(root, "Alpha"))
	mustWrite(t, filepath.Join(root, "b.go"), "package b\n")
	mustWrite(t, filepath.Join(root, "A.go"), "package a\n")
	mustWrite(t, filepath.Join(root, ".hidden"), "hidden\n")
	mustWrite(t, filepath.Join(root, "Alpha", "child.go"), "package child\n")

	explorer := OpenExplorer(root)
	assertExplorerNames(t, explorer, []string{filepath.Base(root), "Alpha", "zeta", "A.go", "b.go"})

	explorer.Select(1)
	explorer.ExpandOrChild()
	assertExplorerNames(t, explorer, []string{filepath.Base(root), "Alpha", "child.go", "zeta", "A.go", "b.go"})
	explorer.ExpandOrChild()
	if selected := explorer.Selected(); selected == nil || selected.Name != "child.go" {
		t.Fatalf("selected after entering expanded directory = %#v", selected)
	}
	explorer.CollapseOrParent()
	if selected := explorer.Selected(); selected == nil || selected.Name != "Alpha" {
		t.Fatalf("selected after moving to parent = %#v", selected)
	}
	explorer.CollapseOrParent()
	assertExplorerNames(t, explorer, []string{filepath.Base(root), "Alpha", "zeta", "A.go", "b.go"})
}

func TestExplorerRefreshRetainsExpansionAndSelection(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "cmd")
	mustMkdir(t, dir)
	mustWrite(t, filepath.Join(dir, "main.go"), "package main\n")

	explorer := OpenExplorer(root)
	explorer.Select(1)
	explorer.ExpandOrChild()
	explorer.ExpandOrChild()
	selected := explorer.Selected()
	if selected == nil || selected.Name != "main.go" {
		t.Fatalf("initial selected node = %#v", selected)
	}

	mustWrite(t, filepath.Join(dir, "app.go"), "package main\n")
	explorer.Refresh()
	selected = explorer.Selected()
	if selected == nil || selected.Name != "main.go" {
		t.Fatalf("selected node after refresh = %#v", selected)
	}
	assertExplorerNames(t, explorer, []string{filepath.Base(root), "cmd", "app.go", "main.go"})
}

func TestExplorerHiddenPolicyAndActivation(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, ".env"), "secret\n")
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n")

	explorer := OpenExplorer(root)
	assertExplorerNames(t, explorer, []string{filepath.Base(root), "main.go"})
	explorer.SetShowHidden(true)
	assertExplorerNames(t, explorer, []string{filepath.Base(root), ".env", "main.go"})
	explorer.Last()
	path, ok := explorer.ActivateSelected()
	if !ok || path != filepath.ToSlash(filepath.Join(root, "main.go")) && path != filepath.Join(root, "main.go") {
		t.Fatalf("activated path = %q, %v", path, ok)
	}
}

func TestExplorerReportsDirectoryFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	explorer := OpenExplorer(path)
	if explorer.Root == nil || explorer.Root.Error == "" || !explorer.Root.Loaded {
		t.Fatalf("failed root = %#v", explorer.Root)
	}
}

func assertExplorerNames(t *testing.T, explorer *Explorer, want []string) {
	t.Helper()
	rows := explorer.Rows()
	got := make([]string, len(rows))
	for i := 0; i < len(rows); i++ {
		got[i] = rows[i].Node.Name
	}
	if len(got) != len(want) {
		t.Fatalf("row names = %#v, want %#v", got, want)
	}
	for i := 0; i < len(got); i++ {
		if got[i] != want[i] {
			t.Fatalf("row names = %#v, want %#v", got, want)
		}
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}
