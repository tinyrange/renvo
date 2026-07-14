package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	manifestPath = "rtg_tests/limitations/manifest.json"
	capturePath  = "rtg_tests/limitations/results.json"
	reportPath   = "LIMITATIONS.html"
	readmePath   = "rtg_tests/README.md"
)

type manifest struct {
	Version int            `json:"version"`
	Policy  string         `json:"policy"`
	Cases   []manifestCase `json:"cases"`
}

type manifestCase struct {
	ID             int    `json:"id"`
	Area           string `json:"area"`
	Title          string `json:"title"`
	Expected       string `json:"expected"`
	CommentaryHTML string `json:"commentary_html"`
	Fixture        string `json:"fixture"`
}

type capture struct {
	Version      int          `json:"version"`
	SourceDigest string       `json:"source_digest"`
	Runs         []captureRun `json:"runs"`
}

type captureRun struct {
	Stage        string        `json:"stage"`
	Target       string        `json:"target"`
	Commit       string        `json:"commit"`
	CapturedAt   string        `json:"captured_at"`
	Observations []observation `json:"observations"`
}

type observation struct {
	ID      int    `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type runSpecs []string

func (r *runSpecs) String() string { return strings.Join(*r, ",") }
func (r *runSpecs) Set(value string) error {
	if !strings.Contains(value, "=") {
		return fmt.Errorf("run must be stage=/path/to/compiler")
	}
	*r = append(*r, value)
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "rtgreport:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("rtgreport", flag.ContinueOnError)
	check := flags.Bool("check", false, "verify generated files and captured results")
	captureResults := flags.Bool("capture", false, "run compilers and replace captured results")
	root := flags.String("root", ".", "repository root")
	target := flags.String("target", "linux/amd64", "target used for captured runs")
	backend := flags.String("backend", "", "backend executable for a host frontend run")
	stdRoot := flags.String("stdroot", "", "standard library root for a host frontend run")
	commit := flags.String("commit", "", "commit recorded for captured runs (defaults to HEAD)")
	var runs runSpecs
	flags.Var(&runs, "run", "compiler run in stage=/path form; repeat for host and stage3")
	if err := flags.Parse(args); err != nil {
		return err
	}
	absRoot, err := filepath.Abs(*root)
	if err != nil {
		return err
	}
	m, err := loadManifest(absRoot)
	if err != nil {
		return err
	}
	if err := validateManifest(absRoot, m); err != nil {
		return err
	}
	if *captureResults {
		if *check {
			return errors.New("-capture and -check cannot be combined")
		}
		if len(runs) == 0 {
			return errors.New("-capture requires at least one -run stage=/path")
		}
		if *commit == "" {
			value, err := gitCommit(absRoot)
			if err != nil {
				return err
			}
			*commit = value
		}
		if *stdRoot == "" {
			*stdRoot = filepath.Join(absRoot, "rtg", "std")
		}
		captured, err := observe(absRoot, m, runs, *target, *commit, *backend, *stdRoot)
		if err != nil {
			return err
		}
		if err := writeJSON(filepath.Join(absRoot, capturePath), captured); err != nil {
			return err
		}
	}
	c, err := loadCapture(absRoot)
	if err != nil {
		return err
	}
	if err := validateCapture(absRoot, m, c); err != nil {
		return err
	}
	report, err := renderReport(m, c)
	if err != nil {
		return err
	}
	readme, err := renderCorpusReadme(absRoot)
	if err != nil {
		return err
	}
	generated := []struct {
		path string
		data []byte
	}{
		{filepath.Join(absRoot, reportPath), report},
		{filepath.Join(absRoot, readmePath), readme},
	}
	for _, item := range generated {
		if *check {
			current, err := os.ReadFile(item.path)
			if err != nil {
				return err
			}
			if !bytes.Equal(current, item.data) {
				return fmt.Errorf("%s is stale; run go run ./cmd/rtgreport", relative(absRoot, item.path))
			}
			continue
		}
		if err := os.WriteFile(item.path, item.data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func loadManifest(root string) (manifest, error) {
	var value manifest
	err := readJSON(filepath.Join(root, manifestPath), &value)
	return value, err
}

func loadCapture(root string) (capture, error) {
	var value capture
	err := readJSON(filepath.Join(root, capturePath), &value)
	return value, err
}

func readJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func validateManifest(root string, m manifest) error {
	if m.Version != 1 || len(m.Cases) == 0 {
		return fmt.Errorf("unsupported or empty limitations manifest")
	}
	seen := make(map[int]bool)
	for i := range m.Cases {
		item := m.Cases[i]
		if item.ID < 1 || seen[item.ID] {
			return fmt.Errorf("invalid or duplicate limitations case id %d", item.ID)
		}
		seen[item.ID] = true
		if item.Area == "" || item.Title == "" || item.CommentaryHTML == "" {
			return fmt.Errorf("limitations case %d is missing human-authored metadata", item.ID)
		}
		switch item.Expected {
		case "accepted", "frontend-diagnostic", "excluded", "backend-failure":
		default:
			return fmt.Errorf("limitations case %d has invalid expected outcome %q", item.ID, item.Expected)
		}
		fixture := filepath.Join(root, filepath.FromSlash(item.Fixture))
		if relative(root, fixture) != filepath.FromSlash(item.Fixture) {
			return fmt.Errorf("limitations case %d fixture escapes repository", item.ID)
		}
		for _, name := range []string{"go.mod", filepath.Join("cmd", "app", "main.go")} {
			if _, err := os.Stat(filepath.Join(fixture, name)); err != nil {
				return fmt.Errorf("limitations case %d fixture: %w", item.ID, err)
			}
		}
	}
	sort.Slice(m.Cases, func(i, j int) bool { return m.Cases[i].ID < m.Cases[j].ID })
	return nil
}

func validateCapture(root string, m manifest, c capture) error {
	if c.Version != 1 || len(c.Runs) == 0 {
		return errors.New("unsupported or empty limitations capture")
	}
	digest, err := sourceDigest(root, m)
	if err != nil {
		return err
	}
	if c.SourceDigest != digest {
		return fmt.Errorf("captured limitations results are stale (source digest %s, want %s); recapture them", c.SourceDigest, digest)
	}
	stages := make(map[string]bool)
	for _, run := range c.Runs {
		if run.Stage == "" || run.Target == "" || run.Commit == "" || run.CapturedAt == "" {
			return errors.New("captured run is missing stage, target, commit, or timestamp")
		}
		if stages[run.Stage] {
			return fmt.Errorf("duplicate captured stage %q", run.Stage)
		}
		stages[run.Stage] = true
		observed := make(map[int]observation)
		for _, item := range run.Observations {
			if _, exists := observed[item.ID]; exists {
				return fmt.Errorf("stage %s has duplicate case %d", run.Stage, item.ID)
			}
			observed[item.ID] = item
		}
		for _, item := range m.Cases {
			got, ok := observed[item.ID]
			if !ok {
				return fmt.Errorf("stage %s is missing case %d", run.Stage, item.ID)
			}
			if !outcomeMatches(item.Expected, got.Status) {
				return fmt.Errorf("stage %s case %d expected %s, observed %s", run.Stage, item.ID, item.Expected, got.Status)
			}
		}
		if len(observed) != len(m.Cases) {
			return fmt.Errorf("stage %s has %d observations for %d manifest cases", run.Stage, len(observed), len(m.Cases))
		}
	}
	if !stages["host"] || !stages["stage3"] || len(stages) != 2 {
		return fmt.Errorf("limitations capture must contain exactly host and stage3 runs")
	}
	return nil
}

func outcomeMatches(expected string, observed string) bool {
	if expected == "excluded" {
		return observed == "frontend-diagnostic"
	}
	return expected == observed
}

func observe(root string, m manifest, specs []string, target string, commit string, backend string, stdRoot string) (capture, error) {
	digest, err := sourceDigest(root, m)
	if err != nil {
		return capture{}, err
	}
	result := capture{Version: 1, SourceDigest: digest}
	capturedAt := time.Now().UTC().Format(time.RFC3339)
	for _, spec := range specs {
		parts := strings.SplitN(spec, "=", 2)
		stage := parts[0]
		compiler, err := filepath.Abs(parts[1])
		if err != nil {
			return capture{}, err
		}
		if stage == "host" && backend == "" {
			return capture{}, errors.New("host capture requires -backend")
		}
		run := captureRun{Stage: stage, Target: target, Commit: commit, CapturedAt: capturedAt}
		for _, item := range m.Cases {
			got := observeCase(root, item, stage, compiler, target, backend, stdRoot)
			run.Observations = append(run.Observations, got)
		}
		result.Runs = append(result.Runs, run)
	}
	return result, nil
}

func observeCase(root string, item manifestCase, stage string, compiler string, target string, backend string, stdRoot string) observation {
	fixture := filepath.Join(root, filepath.FromSlash(item.Fixture))
	temp, err := os.MkdirTemp("", "rtg-limitations-*")
	if err != nil {
		return observation{ID: item.ID, Status: "backend-failure", Message: err.Error()}
	}
	defer os.RemoveAll(temp)
	output := filepath.Join(temp, "app")
	cmd := exec.Command(compiler, "-t", target, "-s", "-o", output, "./cmd/app")
	cmd.Dir = fixture
	cmd.Env = replaceEnv(os.Environ(), "PWD", fixture)
	cmd.Env = replaceEnv(cmd.Env, "RTG_STDROOT", stdRoot)
	if stage == "host" {
		cmd.Env = replaceEnv(cmd.Env, "RTG_BACKEND", backend)
	}
	out, runErr := cmd.CombinedOutput()
	message := normalizeMessage(root, fixture, string(out))
	if runErr == nil {
		return observation{ID: item.ID, Status: "accepted", Message: "compiled successfully"}
	}
	if message == "" {
		message = runErr.Error()
	}
	status := "backend-failure"
	if isFrontendDiagnostic(message) {
		status = "frontend-diagnostic"
	}
	return observation{ID: item.ID, Status: status, Message: message}
}

func replaceEnv(env []string, key string, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	return append(out, prefix+value)
}

func normalizeMessage(root string, fixture string, message string) string {
	message = strings.ReplaceAll(message, filepath.ToSlash(fixture), ".")
	message = strings.ReplaceAll(message, fixture, ".")
	message = strings.ReplaceAll(message, filepath.ToSlash(root), "<repo>")
	message = strings.ReplaceAll(message, root, "<repo>")
	message = strings.ReplaceAll(message, "\x00", `\x00`)
	return strings.TrimSpace(message)
}

func isFrontendDiagnostic(message string) bool {
	if strings.Contains(message, "error RTG-") && !strings.Contains(message, "error RTG-BACKEND-") {
		return true
	}
	markers := []string{
		"frontend pipeline failed", "source error at", "missing module at", "invalid module at",
		"bad package:", "directory read failed:", "file read failed:", "bad build constraint:",
		"source parse failed:", "unresolved import:", "cgo import", "unsupported target:",
	}
	for _, marker := range markers {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func sourceDigest(root string, m manifest) (string, error) {
	var paths []string
	addTree := func(base string, include func(string) bool) error {
		return filepath.WalkDir(filepath.Join(root, base), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == "sandbox" {
					return filepath.SkipDir
				}
				return nil
			}
			rel := relative(root, path)
			if include(rel) {
				paths = append(paths, rel)
			}
			return nil
		})
	}
	entries, err := filepath.Glob(filepath.Join(root, "*.go"))
	if err != nil {
		return "", err
	}
	for _, path := range entries {
		if !strings.HasSuffix(path, "_test.go") {
			paths = append(paths, relative(root, path))
		}
	}
	paths = append(paths, "go.mod", manifestPath)
	if err := addTree("rtg", func(path string) bool {
		return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
	}); err != nil {
		return "", err
	}
	if err := addTree("cmd/rtgreport", func(path string) bool {
		return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
	}); err != nil {
		return "", err
	}
	if err := addTree("rtg_tests/limitations", func(path string) bool {
		return path != capturePath
	}); err != nil {
		return "", err
	}
	sort.Strings(paths)
	hash := sha256.New()
	seen := make(map[string]bool)
	for _, path := range paths {
		path = filepath.Clean(path)
		if seen[path] {
			continue
		}
		seen[path] = true
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(hash, "%s\x00%d\x00", filepath.ToSlash(path), len(data))
		hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func renderReport(m manifest, c capture) ([]byte, error) {
	var out strings.Builder
	out.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<title>RTG Frontend Capability Report</title>\n")
	out.WriteString("<style>body{font-family:system-ui,sans-serif;margin:2rem;line-height:1.35;color:#1f2328}table{border-collapse:collapse;width:100%;margin:1rem 0 1.5rem}th,td{border:1px solid #ccc;padding:.45rem;vertical-align:top}th{background:#f3f3f3;text-align:left}code{font-size:.85rem;white-space:pre-wrap}.meta{color:#555;max-width:70rem}.accepted{color:#116329}.diagnostic{color:#9a6700}.failure{color:#cf222e}</style>\n</head>\n<body>\n")
	out.WriteString("<h1>RTG Frontend Capability Report</h1>\n<p class=\"meta\">")
	out.WriteString(html.EscapeString(m.Policy))
	out.WriteString("</p>\n<p class=\"meta\">This file is generated from checked fixtures, human commentary in <code>")
	out.WriteString(manifestPath)
	out.WriteString("</code>, and machine observations in <code>")
	out.WriteString(capturePath)
	out.WriteString("</code>. Do not edit observed fields here.</p>\n")
	for _, run := range c.Runs {
		counts := make(map[string]int)
		for _, item := range run.Observations {
			counts[item.Status]++
		}
		fmt.Fprintf(&out, "<h2>%s / %s</h2>\n", html.EscapeString(run.Stage), html.EscapeString(run.Target))
		fmt.Fprintf(&out, "<p class=\"meta\">Compiler commit <code>%s</code>; captured <code>%s</code>; source digest <code>%s</code>.</p>\n", html.EscapeString(run.Commit), html.EscapeString(run.CapturedAt), html.EscapeString(c.SourceDigest))
		fmt.Fprintf(&out, "<p class=\"meta\">%d probes: %d accepted, %d frontend diagnostics, %d backend/compiler failures.</p>\n", len(run.Observations), counts["accepted"], counts["frontend-diagnostic"], counts["backend-failure"])
	}
	out.WriteString("<h2>Fixture observations</h2>\n<table><thead><tr><th>#</th><th>Area</th><th>Capability / limitation</th><th>Expected</th><th>Human commentary</th><th>Observed runs</th><th>Fixture</th></tr></thead><tbody>\n")
	observations := make(map[string]map[int]observation)
	for _, run := range c.Runs {
		observations[run.Stage] = make(map[int]observation)
		for _, item := range run.Observations {
			observations[run.Stage][item.ID] = item
		}
	}
	for _, item := range m.Cases {
		fmt.Fprintf(&out, "<tr id=\"case-%03d\"><td>%d</td><td>%s</td><td>%s</td><td><code>%s</code></td><td>%s</td><td>", item.ID, item.ID, html.EscapeString(item.Area), html.EscapeString(item.Title), html.EscapeString(item.Expected), item.CommentaryHTML)
		for index, run := range c.Runs {
			if index != 0 {
				out.WriteString("<br>")
			}
			got := observations[run.Stage][item.ID]
			class := "failure"
			if got.Status == "accepted" {
				class = "accepted"
			} else if got.Status == "frontend-diagnostic" {
				class = "diagnostic"
			}
			fmt.Fprintf(&out, "<strong>%s:</strong> <span class=\"%s\">%s</span> — <code>%s</code>", html.EscapeString(run.Stage), class, html.EscapeString(got.Status), html.EscapeString(got.Message))
		}
		fixtureSource := filepath.ToSlash(filepath.Join(item.Fixture, "cmd", "app", "main.go"))
		fixtureDir := filepath.ToSlash(item.Fixture) + "/"
		fmt.Fprintf(&out, "</td><td><a href=\"%s\">fixture</a> (<a href=\"%s\">main.go</a>)</td></tr>\n", html.EscapeString(fixtureDir), html.EscapeString(fixtureSource))
	}
	out.WriteString("</tbody></table>\n</body>\n</html>\n")
	return []byte(out.String()), nil
}

func renderCorpusReadme(root string) ([]byte, error) {
	quick, err := countModules(filepath.Join(root, "rtg_tests", "quick"))
	if err != nil {
		return nil, err
	}
	extended, err := countModules(filepath.Join(root, "rtg_tests", "extended"))
	if err != nil {
		return nil, err
	}
	regressions, err := countModules(filepath.Join(root, "rtg_tests", "regressions"))
	if err != nil {
		return nil, err
	}
	negative, err := countModules(filepath.Join(root, "rtg_tests", "negative"))
	if err != nil {
		return nil, err
	}
	text := fmt.Sprintf("# RTG Frontend Test Corpus\n\n"+
		"This file is generated by `go run ./cmd/rtgreport` from the checked-in tree.\n\n"+
		"`rtg_tests` is a frontend acceptance corpus kept outside `rtg/` so it can survive a frontend rewrite. Each case is its own Go module directory and must print only `PASS\\n` on success.\n\n"+
		"- `quick/` contains %d tests intended to run on every frontend check.\n"+
		"- `extended/` contains %d broader interaction tests gated by `RTG_FRONTEND_EXTENDED_TESTS=1`.\n"+
		"- `regressions/` contains %d hand-maintained cases that are never replaced by the generator.\n"+
		"- `negative/` contains %d checked rejection cases with exact phase, code, source location, and message expectations.\n"+
		"- `limitations/` contains executable capability probes whose host and stage3 compiler observations generate `LIMITATIONS.html`.\n\n"+
		"`corpus_manifest.json` records case, declared-variant, and normalized AST-shape counts. Tests recompute those fingerprints from the checked tree, so clone count cannot stand in for structural coverage.\n\n"+
		"By default the harness validates that each corpus case is valid host Go and prints `PASS\\n`. If `./rtg/cmd/rtg` exists, the harness builds it with host Go and also checks compiler output. Set `RTG_FRONTEND=/path/to/compiler` to test a specific compiler, such as a stage2 self-hosted binary.\n\n"+
		"The generated corpus is maintained by:\n\n"+
		"```sh\n"+
		"go run ./rtg_tests/generate_tests.go\n"+
		"go run ./cmd/rtgreport\n"+
		"```\n", quick, extended, regressions, negative)
	return []byte(text), nil
}

func countModules(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && entry.Name() == "go.mod" {
			count++
		}
		return nil
	})
	return count, err
}

func gitCommit(root string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func relative(root string, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.Clean(rel)
}
