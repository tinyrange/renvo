package rfe

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const maxArchive = 32 << 20

type Dependency struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256,omitempty"`
}
type Manifest struct {
	Format   int          `json:"format"`
	Name     string       `json:"name"`
	Import   string       `json:"import"`
	Entry    string       `json:"entry,omitempty"`
	Requires []Dependency `json:"requires,omitempty"`
}
type Package struct {
	Manifest Manifest
	Files    map[string][]byte
	Digest   string
}

var packageName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)

func manifest(data []byte) (Manifest, error) {
	var m Manifest
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&m); err != nil {
		return m, err
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return m, fmt.Errorf("trailing manifest content")
	}
	if m.Format != 1 || !packageName.MatchString(m.Name) || m.Import == "" || strings.ContainsAny(m.Import, " \t\n\\\"") || m.Entry != "" && !token.IsIdentifier(m.Entry) {
		return m, fmt.Errorf("invalid RFE manifest")
	}
	seen := map[string]bool{}
	for _, dep := range m.Requires {
		if !packageName.MatchString(dep.Name) || dep.Name == m.Name || seen[dep.Name] {
			return m, fmt.Errorf("invalid or duplicate dependency %q", dep.Name)
		}
		seen[dep.Name] = true
	}
	return m, nil
}
func validName(n string) bool {
	return n != "" && !strings.Contains(n, "\\") && !strings.Contains(n, ":") && !strings.HasPrefix(n, "/") && path.Clean(n) == n && n != ".." && !strings.HasPrefix(n, "../")
}
func Decode(data []byte) (*Package, error) {
	if len(data) > maxArchive {
		return nil, fmt.Errorf("RFE source exceeds size limit")
	}
	if !bytes.HasPrefix(data, []byte("RFE 1\n")) {
		return nil, fmt.Errorf("expected an RFE 1 text archive")
	}
	return decodeText(data)
}

// The canonical source format is an editable text archive. Sections contain
// ordinary Go, the instruction DSL, or the manifest.
func decodeText(data []byte) (*Package, error) {
	p := &Package{Files: map[string][]byte{}}
	lines := bytes.SplitAfter(data, []byte("\n"))
	name := ""
	for _, line := range lines[1:] {
		text := strings.TrimSuffix(string(line), "\n")
		if strings.HasPrefix(text, "-- ") && strings.HasSuffix(text, " --") {
			name = strings.TrimSuffix(strings.TrimPrefix(text, "-- "), " --")
			if !validName(name) || p.Files[name] != nil || len(p.Files) >= 256 {
				return nil, fmt.Errorf("invalid RFE section %q", name)
			}
			p.Files[name] = []byte{}
			continue
		}
		if name == "" {
			if strings.TrimSpace(text) != "" {
				return nil, fmt.Errorf("content before first RFE section")
			}
			continue
		}
		p.Files[name] = append(p.Files[name], line...)
	}
	var err error
	p.Manifest, err = manifest(p.Files["rfe.json"])
	if err != nil {
		return nil, err
	}
	for _, d := range p.Manifest.Requires {
		if d.SHA256 != "" {
			digest, e := hex.DecodeString(d.SHA256)
			if e != nil || len(digest) != 32 {
				return nil, fmt.Errorf("invalid dependency SHA-256")
			}
		}
	}
	digest := sha256.Sum256(data)
	p.Digest = hex.EncodeToString(digest[:])
	return p, nil
}

// Resolve is offline and rejects cycles, hash mismatches and conflicting import
// identities. Returned packages are in deterministic dependency-first order.
func Resolve(root []byte, load func(string) ([]byte, error)) ([]*Package, error) {
	seen, active := map[string]*Package{}, map[string]bool{}
	imports := map[string]string{}
	var result []*Package
	var visit func(*Package) error
	visit = func(p *Package) error {
		name := p.Manifest.Name
		if active[name] {
			return fmt.Errorf("RFE dependency cycle at %s", name)
		}
		if owner, ok := imports[p.Manifest.Import]; ok && owner != name {
			return fmt.Errorf("duplicate RFE import identity %s", p.Manifest.Import)
		}
		imports[p.Manifest.Import] = name
		active[name] = true
		seen[name] = p
		for _, dep := range p.Manifest.Requires {
			child := seen[dep.Name]
			if child == nil {
				data, err := load(dep.Name)
				if err != nil {
					return fmt.Errorf("dependency %s: %w", dep.Name, err)
				}
				child, err = Decode(data)
				if err != nil {
					return fmt.Errorf("dependency %s: %w", dep.Name, err)
				}
			}
			if child.Manifest.Name != dep.Name || dep.SHA256 != "" && child.Digest != dep.SHA256 {
				return fmt.Errorf("dependency hash mismatch: %s", dep.Name)
			}
			if active[dep.Name] {
				return fmt.Errorf("RFE dependency cycle at %s", dep.Name)
			}
			if seen[dep.Name] == nil {
				if err := visit(child); err != nil {
					return err
				}
			}
		}
		active[name] = false
		result = append(result, p)
		return nil
	}
	p, err := Decode(root)
	if err != nil {
		return nil, err
	}
	if err := visit(p); err != nil {
		return nil, err
	}
	return result, nil
}

// Workspace writes only into a newly-created build directory supplied by the
// caller. Imports between RFE packages are relocated together, so archive
// contents, rather than matching packages in the checkout, are compiled.
func Workspace(dir, renvoRoot string, packages []*Package) error {
	if len(packages) == 0 {
		return fmt.Errorf("no RFE packages")
	}
	const module = "renvo.dev/rfe/workspace"
	imports := map[string]string{}
	for _, p := range packages {
		imports[p.Manifest.Import] = module + "/packages/" + p.Manifest.Name
	}
	for _, p := range packages {
		dst := filepath.Join(dir, "packages", p.Manifest.Name)
		if err := os.MkdirAll(dst, 0o755); err != nil {
			return err
		}
		names := make([]string, 0, len(p.Files))
		for n := range p.Files {
			names = append(names, n)
		}
		sort.Strings(names)
		written := map[string]bool{}
		for _, name := range names {
			source := p.Files[name]
			if strings.HasSuffix(name, ".lower") {
				var err error
				source, err = GenerateLowering(source)
				if err != nil {
					return fmt.Errorf("%s/%s: %w", p.Manifest.Name, name, err)
				}
				name = strings.TrimSuffix(name, ".lower") + "_generated.go"
			} else if !strings.HasSuffix(name, ".go") {
				continue
			}
			if strings.Contains(name, "/") || written[name] {
				return fmt.Errorf("RFE v1 requires unique top-level source files")
			}
			written[name] = true
			fs := token.NewFileSet()
			file, err := parser.ParseFile(fs, name, source, parser.ParseComments)
			if err != nil {
				return err
			}
			for _, im := range file.Imports {
				old, err := strconv.Unquote(im.Path.Value)
				if err != nil {
					return err
				}
				if replacement, ok := imports[old]; ok {
					im.Path.Value = strconv.Quote(replacement)
				}
			}
			var formatted bytes.Buffer
			if err = format.Node(&formatted, fs, file); err != nil {
				return err
			}
			if err = os.WriteFile(filepath.Join(dst, name), formatted.Bytes(), 0o644); err != nil {
				return err
			}
		}
	}
	root := packages[len(packages)-1]
	if root.Manifest.Entry != "" {
		main := fmt.Sprintf("package main\nimport (\"os\"; entry %q)\nfunc main(){os.Exit(entry.%s(os.Args[1:]))}\n", imports[root.Manifest.Import], root.Manifest.Entry)
		if err := os.MkdirAll(filepath.Join(dir, "cmd"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "cmd", "main.go"), []byte(main), 0o644); err != nil {
			return err
		}
	}
	goMod := fmt.Sprintf("module %s\n\ngo 1.25.5\n\nrequire renvo.dev v0.0.0\nreplace renvo.dev => %s\n", module, strconv.Quote(renvoRoot))
	return os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644)
}
