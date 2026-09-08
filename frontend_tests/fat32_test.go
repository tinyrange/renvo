package frontend_tests

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"renvo.dev/device/block"
	"renvo.dev/device/fat32"
	"renvo.dev/device/shell"
	rio "renvo.dev/std/io"
	"strings"
	"testing"
	"unicode/utf16"
)

type fatDisk struct {
	data                           []byte
	writes, reads, maxRead, failAt int
}

func (d *fatDisk) Blocks() uint32 { return uint32(len(d.data) / 512) }
func (d *fatDisk) ReadBlocks(lba uint32, b []byte) error {
	if !block.Valid(d.Blocks(), lba, len(b)) {
		return block.ErrRange
	}
	d.reads++
	if len(b) > d.maxRead {
		d.maxRead = len(b)
	}
	copy(b, d.data[int(lba)*512:])
	return nil
}
func (d *fatDisk) WriteBlocks(lba uint32, b []byte) error {
	if !block.Valid(d.Blocks(), lba, len(b)) {
		return block.ErrRange
	}
	d.writes++
	if d.failAt > 0 && d.writes >= d.failAt {
		return errors.New("injected write error")
	}
	copy(d.data[int(lba)*512:], b)
	return nil
}
func (d *fatDisk) Sync() error { return nil }
func newFATDisk() *fatDisk {
	d := &fatDisk{data: make([]byte, (32+2*520+65530*2)*512)}
	b := d.data[:512]
	b[0] = 0xeb
	b[2] = 0x90
	binary.LittleEndian.PutUint16(b[11:], 512)
	b[13] = 2
	binary.LittleEndian.PutUint16(b[14:], 32)
	b[16] = 2
	binary.LittleEndian.PutUint32(b[32:], d.Blocks())
	binary.LittleEndian.PutUint32(b[36:], 520)
	binary.LittleEndian.PutUint32(b[44:], 2)
	binary.LittleEndian.PutUint16(b[48:], 1)
	binary.LittleEndian.PutUint16(b[50:], 6)
	b[510] = 0x55
	b[511] = 0xaa
	copy(d.data[6*512:], b)
	for _, n := range []int{1, 7} {
		f := d.data[n*512:]
		binary.LittleEndian.PutUint32(f, 0x41615252)
		binary.LittleEndian.PutUint32(f[484:], 0x61417272)
		binary.LittleEndian.PutUint32(f[488:], 65529)
		f[510] = 0x55
		f[511] = 0xaa
	}
	for _, sector := range []int{32, 552} {
		f := d.data[sector*512:]
		binary.LittleEndian.PutUint32(f, 0x0ffffff8)
		binary.LittleEndian.PutUint32(f[4:], 0x0fffffff)
		binary.LittleEndian.PutUint32(f[8:], 0x0fffffff)
	}
	return d
}
func readFATFile(t *testing.T, v *fat32.Volume, path string) []byte {
	t.Helper()
	f, err := v.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []byte
	b := make([]byte, 1500)
	for {
		n, err := f.Read(b)
		out = append(out, b[:n]...)
		if err == rio.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}
func TestFAT32ReadWriteDirectories(t *testing.T) {
	d := newFATDisk()
	boot := append([]byte(nil), d.data[:512]...)
	v, err := fat32.Mount(d)
	if err != nil {
		t.Fatal(err)
	}
	if d.writes != 0 {
		t.Fatal("mount wrote media")
	}
	if err = v.Mkdir("/docs"); err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte("hello FAT32\n"), 500)
	if err = v.WriteFile("/docs/readme.txt", payload); err != nil {
		t.Fatal(err)
	}
	if got := readFATFile(t, v, "/DOCS/README.TXT"); !bytes.Equal(got, payload) {
		t.Fatal("content mismatch")
	}
	if d.maxRead < 1024 {
		t.Fatal("file reads were not batched")
	}
	items, err := v.ReadDir("/docs")
	if err != nil || len(items) != 1 || items[0].Size != uint32(len(payload)) {
		t.Fatalf("directory: %v %v", items, err)
	}
	if err = v.Remove("/docs"); err != fat32.ErrNotEmpty {
		t.Fatal("nonempty removal accepted", err)
	}
	if err = v.WriteFile("/docs/readme.txt", []byte("short")); err != nil {
		t.Fatal(err)
	}
	v, err = fat32.Mount(d)
	if err != nil {
		t.Fatal(err)
	}
	if string(readFATFile(t, v, "/docs/readme.txt")) != "short" {
		t.Fatal("replacement lost")
	}
	if err = v.Remove("/docs/readme.txt"); err != nil {
		t.Fatal(err)
	}
	if err = v.Remove("/docs"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(boot, d.data[:512]) {
		t.Fatal("boot changed")
	}
	if !bytes.Equal(d.data[32*512:552*512], d.data[552*512:1072*512]) {
		t.Fatal("FAT copies diverged")
	}
	for _, n := range []int{1, 7} {
		if binary.LittleEndian.Uint32(d.data[n*512+488:]) != 0xffffffff {
			t.Fatal("stale FSInfo")
		}
	}
}
func TestFAT32FailedReplacementPreservesOldFile(t *testing.T) {
	d := newFATDisk()
	v, _ := fat32.Mount(d)
	if err := v.WriteFile("/old.txt", []byte("original")); err != nil {
		t.Fatal(err)
	}
	d.failAt = d.writes + 6
	if err := v.WriteFile("/old.txt", bytes.Repeat([]byte("new"), 1000)); err == nil {
		t.Fatal("expected injected error")
	}
	if string(readFATFile(t, v, "/old.txt")) != "original" {
		t.Fatal("original damaged")
	}
	before := d.writes
	if err := v.Mkdir("/later"); err != fat32.ErrReadOnly || d.writes != before {
		t.Fatal("failed volume allowed writes", err)
	}
}
func TestFAT32RejectGeometry(t *testing.T) {
	for _, offset := range []int{12, 13, 14, 16, 32, 36, 44, 510} {
		d := newFATDisk()
		d.data[offset] = 0
		if _, err := fat32.Mount(d); err == nil {
			t.Errorf("accepted geometry mutation %d", offset)
		}
	}
}
func TestFAT32Clean(t *testing.T) {
	if got := fat32.Clean("/a/b", "../../../../c/./d"); got != "/c/d" {
		t.Fatal(got)
	}
}

func TestFAT32MBRAndDirectoryGrowth(t *testing.T) {
	d := newFATDisk()
	const base = 2048
	data := make([]byte, len(d.data)+base*512)
	copy(data[base*512:], d.data)
	data[450] = 12
	binary.LittleEndian.PutUint32(data[454:], base)
	binary.LittleEndian.PutUint32(data[458:], d.Blocks())
	data[510], data[511] = 0x55, 0xaa
	d.data = data
	mbr := append([]byte(nil), data[:512]...)
	v, err := fat32.Mount(d)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if err := v.WriteFile(fmt.Sprintf("/F%02d.TXT", i), nil); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := v.ReadDir("/")
	if err != nil || len(entries) != 40 {
		t.Fatalf("directory growth: %d %v", len(entries), err)
	}
	if !bytes.Equal(mbr, data[:512]) {
		t.Fatal("MBR changed")
	}
}

func TestFAT32LongNameAndMalformedFallback(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		d := newFATDisk()
		dir := d.data[1072*512:]
		short := []byte("LONGNA~1TXT")
		var sum byte
		for _, c := range short {
			sum = (sum>>1 | sum<<7) + c
		}
		for i := 0; i < 32; i++ {
			dir[i] = 0xff
		}
		dir[0], dir[11], dir[12], dir[13], dir[26], dir[27] = 0x41, 15, 0, sum, 0, 0
		name := "Long name.txt"
		positions := []int{1, 3, 5, 7, 9, 14, 16, 18, 20, 22, 24, 28, 30}
		for i, p := range positions {
			binary.LittleEndian.PutUint16(dir[p:], uint16(name[i]))
		}
		if malformed {
			dir[13]++
		}
		copy(dir[32:43], short)
		dir[43] = 32
		v, err := fat32.Mount(d)
		if err != nil {
			t.Fatal(err)
		}
		items, err := v.ReadDir("/")
		want := name
		if malformed {
			want = "LONGNA~1.TXT"
		}
		if err != nil || len(items) != 1 || items[0].Name != want {
			t.Fatalf("LFN: %v %v", items, err)
		}
		if err := v.Remove("/" + want); err != nil {
			t.Fatal(err)
		}
		if dir[32] != 0xe5 || (!malformed && dir[0] != 0xe5) || (malformed && dir[0] == 0xe5) {
			t.Fatal("incorrect LFN unlink")
		}
	}
}

func TestFAT32DirectoryCycle(t *testing.T) {
	d := newFATDisk()
	for _, table := range []int{32, 552} {
		binary.LittleEndian.PutUint32(d.data[table*512+8:], 2)
	}
	for i := 0; i < 1024; i += 32 {
		d.data[1072*512+i] = 0xe5
	}
	v, err := fat32.Mount(d)
	if err != nil {
		t.Fatal(err)
	}
	_, err = v.ReadDir("/")
	if !fat32.Is(err, fat32.ErrCorrupt) || d.reads > 10 {
		t.Fatalf("cycle: %v, %d reads", err, d.reads)
	}
}

func TestFAT32Shell(t *testing.T) {
	args, err := shell.Split(`write "a.txt" 'hello world' empty\ word ""`)
	if err != nil || !reflect.DeepEqual(args, []string{"write", "a.txt", "hello world", "empty word", ""}) {
		t.Fatalf("split: %q %v", args, err)
	}
	for _, line := range []string{`write "unfinished`, `write trailing\`} {
		if _, err := shell.Split(line); err == nil {
			t.Fatal("accepted incomplete quoting")
		}
	}
	d := newFATDisk()
	v, err := fat32.Mount(d)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	s := shell.New(v, &out)
	s.Execute("mkdir docs")
	s.Execute("cd docs")
	s.Execute(`write note.txt "hello world"`)
	s.Execute("cat note.txt")
	if out.String() != "hello world\r\n" {
		t.Fatalf("output: %q", out.String())
	}
	if err := v.WriteFile("/docs/mixed.txt", []byte("one\r\ntwo\nthree\rfour\x1b[2J")); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	s.Execute("cat mixed.txt")
	if out.String() != "one\r\ntwo\r\nthree\r\nfour.[2J\r\n" {
		t.Fatalf("CR/LF/control handling: %q", out.String())
	}
	if err := v.WriteFile("/docs/lines.txt", []byte(strings.Repeat("line\r\n", 30))); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	s.Execute("head lines.txt")
	if out.String() != strings.Repeat("line\r\n", 20) {
		t.Fatal("head line count")
	}
	s.Execute("cd missing")
	if s.Cwd != "/docs" {
		t.Fatal("failed cd changed directory")
	}
	s.Execute("cd ..")
	if s.Cwd != "/" {
		t.Fatal("parent navigation")
	}
}

func installTestLFN(dir []byte, name, alias string, attr byte) {
	units := utf16.Encode([]rune(name))
	slots := (len(units) + 12) / 13
	var sum byte
	for _, c := range []byte(alias) {
		sum = (sum>>1 | sum<<7) + c
	}
	positions := []int{1, 3, 5, 7, 9, 14, 16, 18, 20, 22, 24, 28, 30}
	for slot := 0; slot < slots; slot++ {
		order := slots - slot
		r := dir[slot*32 : (slot+1)*32]
		r[0], r[11], r[13] = byte(order), 15, sum
		if slot == 0 {
			r[0] |= 64
		}
		for i, p := range positions {
			index := (order-1)*13 + i
			ch := uint16(0xffff)
			if index < len(units) {
				ch = units[index]
			} else if index == len(units) {
				ch = 0
			}
			binary.LittleEndian.PutUint16(r[p:], ch)
		}
	}
	copy(dir[slots*32:], alias)
	dir[slots*32+11] = attr
}

func TestFAT32LFNSlotLengths(t *testing.T) {
	for _, name := range []string{strings.Repeat("a", 13), strings.Repeat("b", 26), strings.Repeat("c", 39), strings.Repeat("d", 255), "a longer name with spaces.txt", "music 🎵.txt"} {
		d := newFATDisk()
		for _, table := range []int{32, 552} {
			binary.LittleEndian.PutUint32(d.data[table*512+8:], 3)
			binary.LittleEndian.PutUint32(d.data[table*512+12:], 0x0fffffff)
		}
		dir := d.data[1072*512:]
		// Start at the last slot in a sector to exercise cross-sector assembly.
		for i := 0; i < 480; i += 32 {
			dir[i] = 0xe5
		}
		installTestLFN(dir[480:], name, "LONGNA~1TXT", 32)
		v, err := fat32.Mount(d)
		if err != nil {
			t.Fatal(err)
		}
		entries, err := v.ReadDir("/")
		if err != nil || len(entries) != 1 || entries[0].Name != name {
			t.Fatalf("LFN %q: %v %v", name, entries, err)
		}
		if _, err := v.Stat("/" + name); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFAT32ShellHiddenAndQuotedNames(t *testing.T) {
	d := newFATDisk()
	dir := d.data[1072*512:]
	installTestLFN(dir, ".metadata", "METADA~1   ", 32)
	copy(dir[64:], "SECRET  TXT")
	dir[75] = 34 // hidden attribute without a leading dot
	installTestLFN(dir[96:], "a friend's notes.txt", "AFRIEN~1TXT", 32)
	v, err := fat32.Mount(d)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	s := shell.New(v, &out)
	s.Execute("ls")
	if out.String() != "'a friend'\\''s notes.txt'  0\r\n" {
		t.Fatalf("default listing: %q", out.String())
	}
	args, err := shell.Split(strings.TrimSuffix(out.String(), "  0\r\n"))
	if err != nil || len(args) != 1 || args[0] != "a friend's notes.txt" {
		t.Fatalf("quoted roundtrip: %q %v", args, err)
	}
	out.Reset()
	s.Execute("ls -la")
	if !strings.Contains(out.String(), ".metadata") || !strings.Contains(out.String(), "SECRET.TXT") {
		t.Fatalf("hidden listing: %q", out.String())
	}
	out.Reset()
	s.Execute("ls -- .metadata")
	if out.String() != ".metadata  0\r\n" {
		t.Fatalf("explicit hidden lookup: %q", out.String())
	}
}
