package embed

import (
	"bytes"
	"testing"
)

func TestNewFSRejectsMalformedArchives(t *testing.T) {
	if _, ok := NewFS("", 1).ReadFileOK("file"); ok {
		t.Fatal("truncated compressed archive was accepted")
	}
	if _, ok := NewFS("\x01x", 1).ReadDirOK("../bad"); ok {
		t.Fatal("invalid path was accepted")
	}
}

func TestEntryAccessors(t *testing.T) {
	entry := Entry{name: "assets", dir: true}
	if entry.Name() != "assets" || !entry.IsDir() {
		t.Fatalf("entry accessors = %q/%v", entry.Name(), entry.IsDir())
	}
}

func TestDecompressArchiveExtendedMatch(t *testing.T) {
	// One literal followed by a distance-one match of 29 bytes. A length-code
	// nibble of 15 selects the following extension byte (29 - 18 = 11).
	archive, ok := decompressArchive("\x01a\x00\x0f\x0b", 30)
	if !ok || !bytes.Equal(archive, bytes.Repeat([]byte{'a'}, 30)) {
		t.Fatalf("extended archive = %q, %v", archive, ok)
	}
	if _, ok := decompressArchive("\x00\x00\x0f", 30); ok {
		t.Fatal("missing extended length byte was accepted")
	}
}

func TestDecompressArchiveBackreferenceBoundaries(t *testing.T) {
	for _, tt := range []struct{ compressed, want string }{
		{"\x07abc\x00\x20", "abcabc"},
		{"\x07abc\x00\x27", "abcabcabcabca"},
		{"\x0fabcd\x00\x30", "abcdabc"},
		{"\x01x\x00\x0f\xff", string(bytes.Repeat([]byte{'x'}, 274))},
	} {
		got, ok := decompressArchive(tt.compressed, len(tt.want))
		if !ok || string(got) != tt.want {
			t.Fatalf("decode %q = %q, %v; want %q", tt.compressed, got, ok, tt.want)
		}
		if _, ok := decompressArchive(tt.compressed, len(tt.want)-1); ok {
			t.Fatalf("accepted backreference past output boundary: %q", tt.compressed)
		}
	}
}

func TestNewFSReadFileOwnsReturnedBytes(t *testing.T) {
	// A single file named "file", containing "abc", encoded as literal groups.
	raw := "\x01\x00\x00\x00\x04\x00\x00\x00\x03\x00\x00\x00fileabc"
	compressed := ""
	for start := 0; start < len(raw); start += 8 {
		end := start + 8
		if end > len(raw) {
			end = len(raw)
		}
		compressed += "\xff" + raw[start:end]
	}
	fs := NewFS(compressed, len(raw))
	first, err := fs.ReadFile("file")
	if err != nil || string(first) != "abc" {
		t.Fatalf("first read = %q, %v", first, err)
	}
	first[0] = 'x'
	second, err := fs.ReadFile("file")
	if err != nil || string(second) != "abc" {
		t.Fatalf("second read = %q, %v", second, err)
	}
}
