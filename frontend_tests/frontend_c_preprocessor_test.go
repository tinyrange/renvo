package frontend_tests

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFrontendCPreprocessorCorpus(t *testing.T) {
	root := repoRoot(t)
	runCPreprocessorCorpus(t, root, frontendCompiler(t, root))
}

func TestFrontendStage3CPreprocessorCorpus(t *testing.T) {
	root := repoRoot(t)
	runCPreprocessorCorpus(t, root, selfHostedFrontendCompiler(t, root))
}

func runCPreprocessorCorpus(t *testing.T, root string, frontend frontendConfig) {
	t.Helper()
	fixture := filepath.Join(root, "frontend_tests", "testdata", "c_preprocessor")
	want, err := os.ReadFile(filepath.Join(fixture, "expected.txt"))
	if err != nil {
		t.Fatal(err)
	}
	command := frontendCommand(frontend,
		"cc", "-E", "-P", "-nostdinc",
		"-I"+filepath.Join(fixture, "first"),
		"-I"+filepath.Join(fixture, "second"),
		"-include", filepath.Join(fixture, "forced.h"),
		filepath.Join(fixture, "main.c"),
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("preprocess checked corpus: %v\n%s", err, output)
	}
	if string(output) != string(want) {
		t.Fatalf("preprocessor token stream:\n got %q\nwant %q", output, want)
	}
}
