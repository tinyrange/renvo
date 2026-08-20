package frontend_tests

import (
	"path/filepath"
	"testing"
)

func TestFrontendStdJSONDynamic(t *testing.T) {
	root := repoRoot(t)
	runFrontendCorpusCase(t, frontendCompiler(t, root), filepath.Join(root, "frontend_tests", "regressions", "std_json_dynamic"))
}
