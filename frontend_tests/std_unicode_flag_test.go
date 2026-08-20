package frontend_tests

import (
	"path/filepath"
	"testing"
)

func TestFrontendStdUnicode(t *testing.T) {
	root := repoRoot(t)
	runFrontendCorpusCase(t, frontendCompiler(t, root), filepath.Join(root, "frontend_tests", "regressions", "std_unicode"))
}
func TestFrontendStdFlag(t *testing.T) {
	root := repoRoot(t)
	runFrontendCorpusCase(t, frontendCompiler(t, root), filepath.Join(root, "frontend_tests", "regressions", "std_flag"))
}
