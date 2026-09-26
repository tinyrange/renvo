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
