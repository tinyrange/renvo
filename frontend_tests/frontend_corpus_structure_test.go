package frontend_tests

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type checkedCorpusManifest struct {
	Version int                  `json:"version"`
	Groups  []checkedCorpusGroup `json:"groups"`
}

type checkedCorpusGroup struct {
	Tier             string         `json:"tier"`
	Group            string         `json:"group"`
	Cases            int            `json:"cases"`
	StructuralShapes int            `json:"structural_shapes"`
	Variants         map[string]int `json:"variants"`
}

func TestFrontendCorpusStructuralCoverage(t *testing.T) {
	root := repoRoot(t)
	data, err := os.ReadFile(filepath.Join(root, "frontend_tests", "corpus_manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest checkedCorpusManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != 1 || len(manifest.Groups) == 0 {
		t.Fatalf("invalid or empty corpus manifest: version=%d groups=%d", manifest.Version, len(manifest.Groups))
	}
	for _, group := range manifest.Groups {
		group := group
		t.Run(group.Tier+"/"+group.Group, func(t *testing.T) {
			dir := filepath.Join(root, "frontend_tests", group.Tier, group.Group)
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			cases := 0
			shapes := make(map[string]bool)
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				caseDir := filepath.Join(dir, entry.Name())
				if _, err := os.Stat(filepath.Join(caseDir, "go.mod")); err != nil {
					continue
				}
				cases++
				shapes[checkedStructuralFingerprint(t, caseDir)] = true
			}
			if cases != group.Cases || len(shapes) != group.StructuralShapes {
				t.Fatalf("manifest drift: cases=%d shapes=%d, want cases=%d shapes=%d; run go run ./frontend_tests/generate_tests.go", cases, len(shapes), group.Cases, group.StructuralShapes)
			}
			if cases >= 5 && group.Group != "legacy_regressions" {
				if len(shapes) < 5 || len(group.Variants) < 5 {
					t.Fatalf("insufficient structural coverage: %d AST shapes, %d declared variants", len(shapes), len(group.Variants))
				}
			}
		})
	}
}

func checkedStructuralFingerprint(t *testing.T, dir string) string {
	t.Helper()
	var paths []string
	if err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	hash := sha256.New()
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		hash.Write([]byte(filepath.Base(path)))
		ast.Inspect(file, func(node ast.Node) bool {
			if node != nil {
				fmt.Fprintf(hash, "%T;", node)
			}
			return true
		})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
