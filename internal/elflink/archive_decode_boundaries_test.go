package elflink

import (
	"bytes"
	"strconv"
	"testing"
)

func TestArchiveHeaderGoldenBytes(t *testing.T) {
	for _, test := range []struct {
		name string
		size int
		want string
	}{
		{"odd.txt/", 3, "odd.txt/        " + "0           " + "0     " + "0     " + "100644  " + "3         " + "`\n"},
		{"#1/25", 1234567890, "#1/25           " + "0           " + "0     " + "0     " + "100644  " + "1234567890" + "`\n"},
		{"123456789012345/", 0, "123456789012345/" + "0           " + "0     " + "0     " + "100644  " + "0         " + "`\n"},
		{"é.o/", 1, "é.o/           " + "0           " + "0     " + "0     " + "100644  " + "1         " + "`\n"},
	} {
		if got := archiveHeader(test.name, test.size); len(got) != 60 || string(got) != test.want {
			t.Errorf("archiveHeader(%q, %d) = %q (%d bytes), want %q (60 bytes)", test.name, test.size, got, len(got), test.want)
		}
	}
}

func TestArchiveGoldenBytes(t *testing.T) {
	members := []Input{{Name: "odd.txt", Data: []byte("abc")}, {Name: "name with spaces", Data: []byte("xy")}}
	got, err := WriteArchive(members)
	if err.Message != "" {
		t.Fatal(err)
	}
	want := "!<arch>\n" +
		"/               " + "0           " + "0     " + "0     " + "100644  " + "4         " + "`\n" + "\x00\x00\x00\x00" +
		"odd.txt/        " + "0           " + "0     " + "0     " + "100644  " + "3         " + "`\n" + "abc\n" +
		"#1/16           " + "0           " + "0     " + "0     " + "100644  " + "18        " + "`\n" + "name with spacesxy"
	if !bytes.Equal(got, []byte(want)) {
		t.Fatalf("archive bytes = %q, want %q", got, want)
	}
	decoded, err := ReadArchive(got)
	if err.Message != "" || len(decoded) != len(members) {
		t.Fatalf("read golden archive: %#v, %#v", decoded, err)
	}
	for i := range members {
		if decoded[i].Name != members[i].Name || !bytes.Equal(decoded[i].Data, members[i].Data) {
			t.Fatalf("golden member %d = %#v, want %#v", i, decoded[i], members[i])
		}
	}
}

func TestArchiveUTF8NameUsesByteWidth(t *testing.T) {
	members := []Input{{Name: "é.o", Data: []byte("abc")}, {Name: "長いファイル名.o", Data: []byte("xy")}}
	data, err := WriteArchive(members)
	if err.Message != "" {
		t.Fatal(err)
	}
	decoded, err := ReadArchive(data)
	if err.Message != "" || len(decoded) != len(members) {
		t.Fatalf("read archive with UTF-8 names: %#v, %#v", decoded, err)
	}
	for i := range members {
		if decoded[i].Name != members[i].Name || !bytes.Equal(decoded[i].Data, members[i].Data) {
			t.Fatalf("UTF-8 member %d = %#v, want %#v", i, decoded[i], members[i])
		}
	}
}

func TestArchiveByteScanBoundaries(t *testing.T) {
	for _, test := range []struct {
		data   string
		prefix string
		want   bool
	}{
		{"", "", true},
		{"", "!<arch>\n", false},
		{"!<arch>", "!<arch>\n", false},
		{"!<arch>\n", "!<arch>\n", true},
		{"!<arch>\ntrailing", "!<arch>\n", true},
		{"!<arch>\x00", "!<arch>\n", false},
		{"\x00<arch>\n", "!<arch>\n", false},
		{"\x7fEL", "\x7fELF", false},
		{"\x7fELF\x00", "\x7fELF", true},
	} {
		if got := archiveHasPrefix([]byte(test.data), test.prefix); got != test.want {
			t.Errorf("archiveHasPrefix(%q, %q) = %v, want %v", test.data, test.prefix, got, test.want)
		}
	}
	for _, test := range []struct {
		data  string
		value byte
		want  int
	}{
		{"", '\n', -1},
		{"\n", '\n', 0},
		{"\nname/\n", '\n', 0},
		{"name/\n", '\n', 5},
		{"name/", '\n', -1},
		{"\x00name/\n", '\n', 6},
		{"name\x00", 0, 4},
	} {
		if got := archiveIndexByte([]byte(test.data), test.value); got != test.want {
			t.Errorf("archiveIndexByte(%q, %d) = %d, want %d", test.data, test.value, got, test.want)
		}
	}
}

func TestArchiveGNUNameScanBoundaries(t *testing.T) {
	for _, test := range []struct {
		name   string
		table  string
		offset int
		want   string
	}{
		{name: "last_byte", table: "long-name.o/\n", want: "long-name.o"},
		{name: "later_entry", table: "a.o/\nlong-name.o/\n", offset: 5, want: "long-name.o"},
		{name: "missing_newline", table: "long-name.o/"},
		{name: "empty_entry", table: "\n"},
		{name: "offset_at_end", table: "long-name.o/\n", offset: 13},
		{name: "empty_table", table: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := appendArchiveMember([]byte("!<arch>\n"), "//", []byte(test.table))
			data = appendArchiveMember(data, "/"+strconv.Itoa(test.offset), []byte("payload"))
			members, err := ReadArchive(data)
			if test.want == "" {
				if err.Message == "" {
					t.Fatal("accepted invalid GNU archive name")
				}
				return
			}
			if err.Message != "" || len(members) != 1 || members[0].Name != test.want || string(members[0].Data) != "payload" {
				t.Fatalf("ReadArchive = %#v, %#v; want member %q with intact payload", members, err, test.want)
			}
		})
	}
}
