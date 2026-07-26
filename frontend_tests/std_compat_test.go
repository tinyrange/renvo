package frontend_tests

import (
	"path/filepath"
	"testing"
)

func TestFrontendStdCompatibility(t *testing.T) {
	root := repoRoot(t)
	corpus := filepath.Join(root, "frontend_tests", "std_compat")
	t.Run("stage0", func(t *testing.T) {
		runFrontendCorpusDirectory(t, corpus, false, frontendCompiler(t, root))
	})
	t.Run("stage3", func(t *testing.T) {
		runFrontendCorpusDirectory(t, corpus, false, selfHostedFrontendCompiler(t, root))
	})
}
