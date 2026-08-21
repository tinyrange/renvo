package frontend_tests

import (
	"path/filepath"
	"testing"
)

func TestFrontendStdTime(t *testing.T) {
	root := repoRoot(t)
	runFrontendCorpusCase(t, frontendCompiler(t, root), filepath.Join(root, "frontend_tests", "regressions", "std_time"))
}
