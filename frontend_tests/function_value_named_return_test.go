package frontend_tests

import (
	"path/filepath"
	"testing"
)

func TestFrontendFunctionValueNamedReturn(t *testing.T) {
	root := repoRoot(t)
	runFrontendCorpusCase(t, frontendCompiler(t, root), filepath.Join(root, "frontend_tests", "regressions", "function_value_named_return"))
}
