package perfgate

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"renvo.dev/internal/rtgprofile"
)

func TestCompilerProfilesFollowPolicy(t *testing.T) {
	p := Load()
	for _, name := range []string{"frontend-linux-amd64", "frontend-wasi-wasm32", "backend-wasi-wasm32"} {
		data, err := os.ReadFile("../../systems/" + name + ".rtg")
		if err != nil {
			t.Fatal(err)
		}
		profile, diagnostic, ok := rtgprofile.Parse(data)
		if !ok {
			t.Fatal(diagnostic)
		}
		if uint64(profile.BinaryLimit) != p.CompilerBytes || uint64(profile.ArenaSize) != p.CompilerArenaBytes {
			t.Errorf("%s: compiler profile differs from shared size/arena policy", name)
		}
	}
}

func TestWindowsArenaOverride(t *testing.T) {
	p := Load()
	for _, target := range p.Targets {
		want := p.CompilerArenaBytes
		if strings.HasPrefix(target.Name, "windows/") {
			want = 256 * 1024 * 1024
		}
		if p.ArenaBytes(target.Name) != want {
			t.Fatalf("wrong compiler arena for %s", target.Name)
		}
	}
	if p.PeakMemoryBytes != 256*1024*1024 {
		t.Fatal("changing Windows arena must not change the process-memory ceiling")
	}
}

func TestPolicyCoversEveryTierOne(t *testing.T) {
	p := Load()
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	var documented []string
	for _, line := range strings.Split(string(readme), "\n") {
		if strings.HasPrefix(line, "| **Tier 1** |") {
			for _, name := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(line, -1) {
				documented = append(documented, name[1])
			}
		}
	}
	var gated []string
	for _, target := range p.Targets {
		gated = append(gated, target.Name)
	}
	slices.Sort(documented)
	slices.Sort(gated)
	if !slices.Equal(documented, gated) {
		t.Fatalf("documented Tier 1 %v differs from required performance matrix %v", documented, gated)
	}
}

func TestPlatformWorkflowsPartitionTierOne(t *testing.T) {
	p := Load()
	seen := map[string]bool{}
	for _, platform := range []string{"linux", "windows", "darwin", "virtual"} {
		targets, err := p.PlatformTargets(platform)
		if err != nil {
			t.Fatal(err)
		}
		for _, target := range targets {
			if seen[target.Name] {
				t.Fatalf("duplicate platform coverage for %s", target.Name)
			}
			seen[target.Name] = true
		}
	}
	if len(seen) != len(p.Targets) {
		t.Fatal("platform workflows do not cover every Tier 1")
	}
	if _, err := p.PlatformTargets("typo"); err == nil {
		t.Fatal("invalid platform accepted")
	}
}

func TestRegressionAndAbsoluteGates(t *testing.T) {
	p := Load()
	sample := Sample{CPU: 1000, Wall: 100000, Memory: 100000, MemoryMetric: "peak_rss", Artifact: 100000, VMSteps: 1000, VMMemory: 10000}
	base := []Sample{sample, sample, sample}
	if fail := p.Check(base, base, true); len(fail) != 0 {
		t.Fatal(fail)
	}
	for _, metric := range []string{"cpu", "memory", "artifact", "steps", "missing", "absolute-outlier", "kind"} {
		t.Run(metric, func(t *testing.T) {
			candidate := slices.Clone(base)
			for i := range candidate {
				switch metric {
				case "cpu":
					candidate[i].CPU = 1251
				case "memory":
					candidate[i].Memory = 120001
				case "artifact":
					candidate[i].Artifact = 110001
				case "steps":
					candidate[i].VMSteps = 1201
				case "missing":
					candidate[i].Memory = 0
				case "kind":
					candidate[i].MemoryMetric = "peak_job_commit"
				}
			}
			if metric == "absolute-outlier" {
				candidate[0].Memory = p.PeakMemoryBytes + 1
			}
			if fail := p.Check(base, candidate, true); len(fail) == 0 {
				t.Fatal("regression passed")
			}
		})
	}
	if failures := p.Check(base, base[:2], false); len(failures) == 0 {
		t.Fatal("missing sample passed")
	}
}

func TestMedianNotBestAndWallIsTelemetry(t *testing.T) {
	p := Load()
	base := []Sample{{CPU: 100, Memory: 100, MemoryMetric: "peak_rss", Artifact: 100}, {CPU: 100, Memory: 100, MemoryMetric: "peak_rss", Artifact: 100}, {CPU: 100, Memory: 100, MemoryMetric: "peak_rss", Artifact: 100}}
	candidate := slices.Clone(base)
	for i := range candidate {
		candidate[i].Wall = 1000000000
	}
	if failures := p.Check(base, candidate, false); len(failures) != 0 {
		t.Fatal(failures)
	}
	candidate[0].CPU = 1
	candidate[1].CPU = 200
	candidate[2].CPU = 300
	if failures := p.Check(base, candidate, false); len(failures) == 0 {
		t.Fatal("best-of-three hid a CPU regression")
	}
}
