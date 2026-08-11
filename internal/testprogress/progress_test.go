package testprogress

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

type recordingLogger struct {
	mu   sync.Mutex
	logs []string
}

func (*recordingLogger) Helper() {}

func (l *recordingLogger) Logf(format string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, fmt.Sprintf(format, args...))
}

func (l *recordingLogger) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.logs, "\n")
}

func TestFilter(t *testing.T) {
	names := []string{"tests/alpha.go", "tests/beta.go"}
	selected, err := Filter(names, `alpha|missing`)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 1 || selected[0] != names[0] {
		t.Fatalf("selected = %#v", selected)
	}
	if _, err := Filter(names, `[`); err == nil {
		t.Fatal("invalid expression unexpectedly accepted")
	}
}

func TestCaseGroup(t *testing.T) {
	for name, want := range map[string]string{
		"tests/append_growth.go":       "append",
		"defer_panic_recover/001_case": "defer_panic_recover",
		"single":                       "single",
	} {
		if got := caseGroup(name); got != want {
			t.Errorf("caseGroup(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestProgressCloseReportsCasesAndGroups(t *testing.T) {
	logger := new(recordingLogger)
	progress := New(logger, "sample corpus", 2)
	progress.Begin("tests/append_one.go")()
	progress.Begin("tests/append_two.go")()
	progress.Close()
	progress.Close()

	logs := logger.String()
	for _, want := range []string{
		"sample corpus: starting 2 cases",
		"sample corpus: completed 2/2 cases",
		"slowest cases:",
		"slowest groups: append=",
	} {
		if !strings.Contains(logs, want) {
			t.Errorf("progress log %q does not contain %q", logs, want)
		}
	}
}
