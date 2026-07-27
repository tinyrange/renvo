package targetinfo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type registrySourceDescriptor struct {
	Name            string   `json:"name"`
	Backend         string   `json:"backend"`
	Aliases         []string `json:"aliases"`
	OS              string   `json:"os"`
	ISA             string   `json:"isa"`
	WordBits        int      `json:"word_bits"`
	Endian          string   `json:"endian"`
	ABI             string   `json:"abi"`
	Image           string   `json:"image"`
	Runtime         []string `json:"runtime"`
	Tags            []string `json:"tags"`
	Advertised      bool     `json:"advertised"`
	Virtual         bool     `json:"virtual"`
	DefaultArena    int      `json:"default_arena"`
	ReleaseArtifact string   `json:"release_artifact"`
	IDE             bool     `json:"ide"`
}

func TestGeneratedRegistryMatchesSource(t *testing.T) {
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
		want := Descriptor{
			Name:            source[i].Name,
			Backend:         source[i].Backend,
			Aliases:         source[i].Aliases,
			OS:              source[i].OS,
			ISA:             source[i].ISA,
			WordBits:        source[i].WordBits,
			Endian:          source[i].Endian,
			ABI:             source[i].ABI,
			Image:           source[i].Image,
			Runtime:         source[i].Runtime,
			Tags:            source[i].Tags,
			Advertised:      source[i].Advertised,
			Virtual:         source[i].Virtual,
			DefaultArena:    source[i].DefaultArena,
			ReleaseArtifact: source[i].ReleaseArtifact,
			IDE:             source[i].IDE,
		}
		if !reflect.DeepEqual(actual[i], want) {
			t.Fatalf("descriptor %d drifted\ngot:  %#v\nwant: %#v", i, actual[i], want)
		}
	}
}

func TestRegistryResultsAreImmutableCopies(t *testing.T) {
	all := All()
	all[0].Name = "changed"
	all[0].Tags[0] = "changed"
	descriptor, ok := Lookup(DefaultName)
	if !ok {
		t.Fatal("default target disappeared")
	}
	if descriptor.Name != DefaultName || descriptor.Tags[0] != "linux" {
		t.Fatalf("registry was mutated through All: %#v", descriptor)
	}
	descriptor.Tags[0] = "changed-again"
	descriptor, _ = Lookup(DefaultName)
	if descriptor.Tags[0] != "linux" {
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
