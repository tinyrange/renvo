//go:build linux && amd64

package elflink

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

func TestArchiveNamesIndexAndBounds(t *testing.T) {
	obj := minimalMainObject()
	members := []Input{{Name: "odd.txt", Data: []byte("abc")}, {Name: "a very long object name.o", Data: obj}}
	data, e := WriteArchive(members)
	if e.Message != "" {
		t.Fatal(e)
	}
	again, e := WriteArchive(members)
	if e.Message != "" || !bytes.Equal(data, again) {
		t.Fatal("nondeterministic archive", e)
	}
	// Independently inspect the big-endian GNU index, whose offset points at
	// a BSD extended-name member header, not at its ELF payload.
	if binary.BigEndian.Uint32(data[68:]) != 1 || string(data[76:81]) != "main\x00" {
		t.Fatalf("index %x", data[68:82])
	}
	at := int(binary.BigEndian.Uint32(data[72:]))
	if string(data[at:at+5]) != "#1/25" {
		t.Fatalf("index points at %q", data[at:at+16])
	}
	decoded, e := ReadArchive(data)
	if e.Message != "" || len(decoded) != 2 {
		t.Fatalf("%v %v", decoded, e)
	}
	for i := range members {
		if decoded[i].Name != members[i].Name || !bytes.Equal(decoded[i].Data, members[i].Data) {
			t.Fatalf("member %d mismatch", i)
		}
	}
	// A separate hand-built GNU string table exercises the other long-name form.
	header := func(name string, n int) string {
		return fmt.Sprintf("%-16s%-12d%-6d%-6d%-8s%-10d`\n", name, 0, 0, 0, "100644", n)
	}
	table := "gnu-long-object-name.o/\n"
	gnu := []byte("!<arch>\n" + header("//", len(table)) + table)
	if len(table)%2 != 0 {
		gnu = append(gnu, '\n')
	}
	gnu = append(gnu, []byte(header("/0", len(obj)))...)
	gnu = append(gnu, obj...)
	decoded, e = ReadArchive(gnu)
	if e.Message != "" || len(decoded) != 1 || decoded[0].Name != "gnu-long-object-name.o" || !bytes.Equal(decoded[0].Data, obj) {
		t.Fatalf("GNU %v %v", decoded, e)
	}
	for _, bad := range [][]byte{data[:7], data[:67], data[:len(data)-1], []byte("!<arch>\n" + header("x/", 999) + "x"), []byte("!<arch>\n" + header("#1/9", 2) + "ab"), []byte("!<arch>\n" + header("/8", 0))} {
		if _, e := ReadArchive(bad); e.Message == "" {
			t.Fatal("accepted malformed archive")
		}
	}
}
