package embed

import "testing"

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
