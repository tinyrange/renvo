package frontend_tests

import (
	"path/filepath"
	"testing"
)

func TestFrontendStdErrors(t *testing.T) {
	root := repoRoot(t)
	runFrontendCorpusCase(t, frontendCompiler(t, root), filepath.Join(root, "frontend_tests", "regressions", "std_errors"))
}
