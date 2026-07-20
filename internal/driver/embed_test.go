package driver

import (
	"bytes"
	"strings"
	"testing"

	"renvo.dev/internal/load"
	renvoembed "renvo.dev/std/embed"
)

type embedMemorySourceFS struct {
	files []load.SourceFile
}

type embedRejectHiddenReadFS struct{ embedMemorySourceFS }

func (fs embedRejectHiddenReadFS) ReadDir(path string) ([]DirEntry, bool) {
	clean := load.CleanPath(path)
	if strings.Contains(clean, "/.cache") || strings.Contains(clean, "/scratch") {
		return nil, false
	}
	return fs.embedMemorySourceFS.ReadDir(path)
}

func (fs embedMemorySourceFS) ReadFile(path string) ([]byte, bool) {
	path = load.CleanPath(path)
	for i := 0; i < len(fs.files); i++ {
		if load.CleanPath(fs.files[i].Path) == path {
			return fs.files[i].Src, true
		}
	}
	return nil, false
}

func (fs embedMemorySourceFS) ReadDir(path string) ([]DirEntry, bool) {
	path = load.CleanPath(path)
	prefix := path
	if prefix != "/" {
		prefix += "/"
	}
	var entries []DirEntry
	for i := 0; i < len(fs.files); i++ {
		filePath := load.CleanPath(fs.files[i].Path)
		if !strings.HasPrefix(filePath, prefix) {
			continue
		}
		rest := filePath[len(prefix):]
		if rest == "" {
			continue
		}
		name := rest
		isDir := false
		if slash := strings.IndexByte(rest, '/'); slash >= 0 {
			name = rest[:slash]
			isDir = true
		}
		found := false
		for j := 0; j < len(entries); j++ {
			if entries[j].Name == name {
				entries[j].IsDir = entries[j].IsDir || isDir
				found = true
				break
			}
		}
		if !found {
			entries = append(entries, DirEntry{Name: name, IsDir: isDir})
		}
	}
	return entries, len(entries) > 0
}

func (fs embedMemorySourceFS) PathExists(path string) bool {
	_, ok := fs.ReadFile(path)
	return ok
}

func sourceEmbedTestFS() embedMemorySourceFS {
	return embedMemorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/go.mod", Src: []byte("module example.com/app\n")},
		{Path: "/repo/app/cmd/app/main.go", Src: []byte("package main\n")},
		{Path: "/repo/app/cmd/app/assets/message.txt", Src: []byte("hello embed\n")},
		{Path: "/repo/app/cmd/app/assets/nested/data.bin", Src: []byte{0, 1, 2, 255}},
		{Path: "/repo/app/cmd/app/assets/.hidden", Src: []byte("hidden")},
		{Path: "/repo/app/cmd/app/assets/_private", Src: []byte("private")},
	}}
}

func TestExpandSourceEmbedsInitializesSupportedVariableTypes(t *testing.T) {
	fs := sourceEmbedTestFS()
	src := []byte(`package main

import emb "embed"

//go:embed assets/message.txt
var message string

//go:embed assets/nested/data.bin
var data []byte

//go:embed assets
var files emb.FS
`)
	expanded, ok, offset, detail := expandSourceEmbeds(fs, "/repo/app/cmd/app/main.go", "/repo/app", src)
	if !ok {
		t.Fatalf("expandSourceEmbeds failed at %d: %s", offset, detail)
	}
	text := string(expanded)
	if !strings.Contains(text, `var message string = "hello embed\x0a"`) {
		t.Fatalf("string initializer missing:\n%s", text)
	}
	if !strings.Contains(text, `var data []byte = []byte("\x00\x01\x02\xff")`) {
		t.Fatalf("byte initializer missing:\n%s", text)
	}
	if !strings.Contains(text, "var files emb.FS = emb.NewFS(") {
		t.Fatalf("FS initializer missing:\n%s", text)
	}
}

func TestExpandSourceEmbedsSupportsGroupedVariableSpecs(t *testing.T) {
	fs := sourceEmbedTestFS()
	src := []byte(`package main

import _ "embed"

var (
	//go:embed assets/message.txt
	message string
)
`)
	expanded, ok, offset, detail := expandSourceEmbeds(fs, "/repo/app/cmd/app/main.go", "/repo/app", src)
	if !ok {
		t.Fatalf("expandSourceEmbeds failed at %d: %s", offset, detail)
	}
	if !strings.Contains(string(expanded), `message string = "hello embed\x0a"`) {
		t.Fatalf("grouped initializer missing:\n%s", expanded)
	}
}

func TestSourceEmbedArchiveRoundTripAndDirectoryRules(t *testing.T) {
	fs := sourceEmbedTestFS()
	files, ok, detail := resolveSourceEmbedPatterns(fs, "/repo/app/cmd/app", "/repo/app", []string{"assets"})
	if !ok {
		t.Fatalf("resolveSourceEmbedPatterns failed: %s", detail)
	}
	if len(files) != 2 || files[0].name != "assets/message.txt" || files[1].name != "assets/nested/data.bin" {
		t.Fatalf("ordinary directory match = %#v", files)
	}

	archive := buildSourceEmbedArchive(files)
	compressed := compressSourceEmbedArchive(archive)
	if len(compressed) >= len(archive) {
		t.Fatalf("archive did not compress: compressed=%d raw=%d", len(compressed), len(archive))
	}
	embedded := renvoembed.NewFS(string(compressed), len(archive))
	message, err := embedded.ReadFile("assets/message.txt")
	if err != nil || !bytes.Equal(message, []byte("hello embed\n")) {
		t.Fatalf("round-trip message = %q, %v", message, err)
	}
	entries, err := embedded.ReadDir("assets")
	if err != nil || len(entries) != 2 || entries[0].Name() != "message.txt" || entries[0].IsDir() || entries[1].Name() != "nested" || !entries[1].IsDir() {
		t.Fatalf("round-trip directory = %#v, %v", entries, err)
	}

	allFiles, ok, detail := resolveSourceEmbedPatterns(fs, "/repo/app/cmd/app", "/repo/app", []string{"all:assets"})
	if !ok || detail != "" || len(allFiles) != 4 {
		t.Fatalf("all: directory match = %#v, %v, %q", allFiles, ok, detail)
	}
}

func TestSourceEmbedPrunesHiddenDirectoriesBeforeWalking(t *testing.T) {
	fs := embedRejectHiddenReadFS{embedMemorySourceFS{files: []load.SourceFile{
		{Path: "/repo/app/cmd/app/assets/message.txt", Src: []byte("message")},
		{Path: "/repo/app/cmd/app/.cache/large.bin", Src: []byte("ignored")},
		{Path: "/repo/app/cmd/app/scratch/large.bin", Src: []byte("unmatched")},
	}}}
	files, ok, detail := resolveSourceEmbedPatterns(fs, "/repo/app/cmd/app", "/repo/app", []string{"assets"})
	if !ok || detail != "" || len(files) != 1 || files[0].name != "assets/message.txt" {
		t.Fatalf("hidden directory affected embed walk: %#v, %v, %q", files, ok, detail)
	}
}

func TestExpandSourceEmbedsReportsMissingImportAndPattern(t *testing.T) {
	fs := sourceEmbedTestFS()
	missingImport := []byte("package main\n//go:embed assets/message.txt\nvar message string\n")
	if _, ok, _, detail := expandSourceEmbeds(fs, "/repo/app/cmd/app/main.go", "/repo/app", missingImport); ok || detail != "embed" {
		t.Fatalf("missing import result = %v, %q", ok, detail)
	}

	missingPattern := []byte("package main\nimport _ \"embed\"\n//go:embed absent.txt\nvar message string\n")
	if _, ok, _, detail := expandSourceEmbeds(fs, "/repo/app/cmd/app/main.go", "/repo/app", missingPattern); ok || detail != "absent.txt" {
		t.Fatalf("missing pattern result = %v, %q", ok, detail)
	}
}

func TestSourceEmbedDirectivesIgnoreStringAndBlockCommentText(t *testing.T) {
	src := []byte("package main\nvar raw = `\n//go:embed raw.txt\n`\n/*\n//go:embed comment.txt\n*/\nvar interpreted = \"//go:embed string.txt\"\n")
	directives, ok, offset := parseSourceEmbedDirectives(src)
	if !ok || offset != 0 || len(directives) != 0 {
		t.Fatalf("non-comment directives = %#v, %v, %d", directives, ok, offset)
	}

	empty := []byte("package main\n//go:embed\nvar value string\n")
	if _, ok, offset := parseSourceEmbedDirectives(empty); ok || offset != len("package main\n") {
		t.Fatalf("empty directive result = %v, %d", ok, offset)
	}
}
