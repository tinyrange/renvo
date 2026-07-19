package frontend_tests

import (
	"os"
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
		if os.Getenv(selfHostTestsEnv) != "1" {
			t.Skipf("set %s=1 to run self-hosted std compatibility", selfHostTestsEnv)
		}
		runFrontendCorpusDirectory(t, corpus, false, selfHostedFrontendCompiler(t, root))
	})
}
