package targetinfo

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"renvo.dev/internal/rtg"
)

type registrySourceDescriptor struct {
	Name            string `json:"name"`
	Backend         string `json:"backend"`
	Advertised      bool   `json:"advertised"`
	Virtual         bool   `json:"virtual"`
	DefaultArena    int    `json:"default_arena"`
	ReleaseArtifact string `json:"release_artifact"`
	IDE             bool   `json:"ide"`
}

func TestGeneratedRegistryMatchesPolicy(t *testing.T) {
	data, err := os.ReadFile("targets.json")
	if err != nil {
		t.Fatal(err)
	}
	var source []registrySourceDescriptor
	if err := json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	actual := All()
	if len(actual) != len(source) {
		t.Fatalf("generated descriptor count = %d, source count = %d", len(actual), len(source))
	}
	for i := 0; i < len(source); i++ {
		got := actual[i]
		want := source[i]
		if got.Name != want.Name || got.Backend != want.Backend ||
			got.Advertised != want.Advertised || got.Virtual != want.Virtual ||
			got.ReleaseArtifact != want.ReleaseArtifact || got.IDE != want.IDE {
			t.Fatalf("descriptor %d policy drifted\ngot:  %#v\nwant: %#v", i, got, want)
		}
	}
}

func TestGeneratedRegistryMatchesMachineDefinitions(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "backend", "definitions", "*.rtg"))
	if err != nil {
		t.Fatal(err)
	}
	machines := make(map[string]rtg.TargetDescriptor)
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.HasPrefix(bytes.TrimSpace(source), []byte("definition ")) {
			continue
		}
		parsed := rtg.ParseImports(source, path, registryImportLoader{})
		if !parsed.Ok {
			t.Fatalf("%s: %#v", path, parsed.Diagnostics)
		}
		hasTarget := false
		for i := 0; i < len(parsed.Declarations); i++ {
			if parsed.Declarations[i].Kind == rtg.DeclTarget {
				hasTarget = true
				break
			}
		}
		if !hasTarget {
			continue
		}
		resolved := rtg.ResolveDefinitions(parsed)
		if !resolved.Ok {
			t.Fatalf("%s: %#v", path, resolved.Diagnostics)
		}
		for _, target := range resolved.Targets {
			machines[target.Descriptor.Name] = target.Descriptor
		}
	}
	for _, got := range All() {
		machine, ok := machines[got.Name]
		if !ok {
			t.Errorf("%s has no machine definition", got.Name)
			continue
		}
		delete(machines, got.Name)
		runtime := append([]string(nil), machine.RuntimeOps...)
		if stringSliceContains(machine.Capabilities, "hosted") {
			runtime = append(runtime, "hosted")
		}
		if !reflect.DeepEqual(got.Aliases, machine.Aliases) ||
			got.Family != machine.Family ||
			got.OS != machine.OS || got.ISA != machine.ISA ||
			got.WordBits != machine.WordBits || got.PointerBits != machine.PointerBits ||
			got.CodePointerBits != machine.CodePointerBits ||
			got.FunctionPointerBits != machine.FunctionPointerBits ||
			got.MaxAlign != machine.MaxAlign ||
			got.Endian != machine.Endian || got.ABI != machine.ABI ||
			got.Image != machine.OutputKind ||
			!reflect.DeepEqual(got.Runtime, runtime) ||
			!reflect.DeepEqual(got.Tags, machine.BuildTags) ||
			!reflect.DeepEqual(got.Capabilities, machine.Capabilities) ||
			got.Definition != machine.Definition ||
			got.DescriptorVersion != machine.Version ||
			got.DefaultArena != machine.ArenaDefault {
			t.Errorf("%s machine projection drifted\ngot:  %#v\nwant: %#v", got.Name, got, machine)
		}
	}
	for name := range machines {
		t.Errorf("%s has no target registry entry", name)
	}
}

type registryImportLoader struct{}

func (registryImportLoader) LoadImport(
	importingFilename string, importPath string,
) rtg.ImportSource {
	path := filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath))
	source, err := os.ReadFile(path)
	return rtg.ImportSource{Source: source, Filename: path, Ok: err == nil}
}

func stringSliceContains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func TestRegistryResultsAreImmutableCopies(t *testing.T) {
	before, ok := Lookup(DefaultName)
	if !ok {
		t.Fatal("default target disappeared")
	}
	all := All()
	all[0].Name = "changed"
	all[0].Tags[0] = "changed"
	all[0].Capabilities[0] = "changed"
	all[0].Definition[0] ^= 0xff
	descriptor, _ := Lookup(DefaultName)
	if !reflect.DeepEqual(descriptor, before) {
		t.Fatalf("registry was mutated through All: %#v", descriptor)
	}
	descriptor.Tags[0] = "changed-again"
	descriptor, _ = Lookup(DefaultName)
	if !reflect.DeepEqual(descriptor, before) {
		t.Fatalf("registry was mutated through Lookup: %#v", descriptor)
	}
}

func TestReleaseWorkflowContainsRegistryArtifacts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(data)
	for _, descriptor := range All() {
		if descriptor.ReleaseArtifact == "" {
			continue
		}
		if !strings.Contains(workflow, "-t "+descriptor.Name+" ") {
			t.Errorf("release workflow does not build %s", descriptor.Name)
		}
		if !strings.Contains(workflow, "dist/"+descriptor.ReleaseArtifact) {
			t.Errorf("release workflow does not publish %s", descriptor.ReleaseArtifact)
		}
	}
}

func TestTargetDocumentationContainsAdvertisedRegistry(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "README.md"),
		filepath.Join("..", "..", "GUIDE.md"),
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		document := string(data)
		for _, descriptor := range All() {
			if descriptor.Advertised && !strings.Contains(document, "`"+descriptor.Name+"`") {
				t.Errorf("%s does not document advertised target %s", path, descriptor.Name)
			}
		}
	}
}

func TestGeneratedMachineDefinitionDocumentationIsComplete(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "backend", "docs", "machine-definitions.generated.md"))
	if err != nil {
		t.Fatal(err)
	}
	document := string(data)
	for _, descriptor := range All() {
		if !strings.Contains(document, "`"+descriptor.Name+"`") {
			t.Errorf("generated documentation is missing target %s", descriptor.Name)
		}
		hash := rtg.HashText(descriptor.Definition)
		if !strings.Contains(document, "`"+hash+"`") {
			t.Errorf("generated documentation is missing definition hash for %s", descriptor.Name)
		}
	}
}
