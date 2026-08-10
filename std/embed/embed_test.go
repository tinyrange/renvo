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
