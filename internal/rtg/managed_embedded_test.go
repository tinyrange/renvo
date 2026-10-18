package rtg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Standalone RBEs intentionally embed the collector. Catch drift so a collector
// correctness fix cannot silently leave one architecture on an older version.
func TestManagedEmbeddedCollectorConsistency(t *testing.T) {
	read := func(path string) string {
		t.Helper()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	core := read("../../backend/definitions/managed_heap64.rtg")
	core = core[strings.Index(core, "go backend {"):]
	var rows []struct {
		Backend string `json:"backend"`
	}
	if err := json.Unmarshal([]byte(read("../../tools/qemu-smoke/targets.json")), &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Backend == "" {
			continue
		}
		t.Run(filepath.Base(row.Backend), func(t *testing.T) {
			source := read(filepath.Join("../..", row.Backend))
			if !strings.Contains(source, core) {
				t.Fatal("standalone collector differs from backend/definitions/managed_heap64.rtg; synchronize the embedded source")
			}
		})
	}
}
