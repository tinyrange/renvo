package main

import (
	"renvo.dev/device/block"
	"renvo.dev/device/fat32"
	"renvo.dev/device/shell"
)

// Sparse backing keeps a standards-sized FAT32 geometry without allocating a
// 34 MiB test disk. All writes still pass through the public block interface.
type disk struct {
	keys    []uint32
	sectors []*sector
}
type sector struct{ data [512]byte }

func (d *disk) Blocks() uint32 { return 70000 }
func (d *disk) Sync() error    { return nil }
func (d *disk) ReadBlocks(lba uint32, b []byte) error {
	if !block.Valid(d.Blocks(), lba, len(b)) {
		return block.ErrRange
	}
	for i := range b {
		b[i] = 0
	}
	for offset := 0; offset < len(b); offset += 512 {
		for i, key := range d.keys {
			if key == lba+uint32(offset/512) {
				copy(b[offset:offset+512], d.sectors[i].data[:])
				break
			}
		}
	}
	return nil
}
func (d *disk) WriteBlocks(lba uint32, b []byte) error {
	if !block.Valid(d.Blocks(), lba, len(b)) {
		return block.ErrRange
	}
	for offset := 0; offset < len(b); offset += 512 {
		key := lba + uint32(offset/512)
		index := -1
		for i, k := range d.keys {
			if k == key {
				index = i
				break
			}
		}
		if index < 0 {
			index = len(d.keys)
			d.keys = append(d.keys, key)
			d.sectors = append(d.sectors, &sector{})
		}
		copy(d.sectors[index].data[:], b[offset:offset+512])
	}
	return nil
}
func put(b []byte, p int, v uint32) {
	for i := 0; i < 4; i++ {
		b[p+i] = byte(v >> uint(i*8))
	}
}

type output struct{ text string }

func (o *output) Write(b []byte) (int, error) { o.text += string(b); return len(b), nil }
func run() bool {
	d := &disk{}
	b := make([]byte, 512)
	b[0] = 0xeb
	b[12] = 2
	b[13] = 1
	b[14] = 32
	b[16] = 2
	put(b, 32, 70000)
	put(b, 36, 550)
	put(b, 44, 2)
	b[510] = 85
	b[511] = 170
	d.WriteBlocks(0, b)
	for i := range b {
		b[i] = 0
	}
	put(b, 0, 0x0ffffff8)
	put(b, 4, 0x0fffffff)
	put(b, 8, 0x0fffffff)
	d.WriteBlocks(32, b)
	d.WriteBlocks(582, b)
	for i := range b {
		b[i] = 0
	}
	// Two full LFN slots have no NUL terminator. Padding outside those slots
	// must not become part of the name on either stage0 or self-hosted targets.
	longName := "abcdefghijklmnopqrstuvwxyz"
	alias := "LONGNA~1TXT"
	var sum byte
	for i := 0; i < len(alias); i++ {
		sum = (sum>>1 | sum<<7) + alias[i]
	}
	positions := [13]int{1, 3, 5, 7, 9, 14, 16, 18, 20, 22, 24, 28, 30}
	for slot := 0; slot < 2; slot++ {
		order := 2 - slot
		at := slot * 32
		b[at] = byte(order)
		if slot == 0 {
			b[at] |= 64
		}
		b[at+11], b[at+13] = 15, sum
		for i, p := range positions {
			b[at+p] = longName[(order-1)*13+i]
		}
	}
	copy(b[64:75], []byte(alias))
	b[75] = 34 // hidden FAT flag, not a dot name
	d.WriteBlocks(1132, b)
	v, err := fat32.Mount(d)
	if err != nil {
		return false
	}
	root, err := v.ReadDir("/")
	if err != nil || len(root) != 1 || root[0].Name != longName || !root[0].IsHidden() {
		return false
	}
	hiddenOut := &output{}
	hiddenShell := shell.New(v, hiddenOut)
	hiddenShell.Execute("ls")
	if hiddenOut.text != "" {
		return false
	}
	hiddenShell.Execute("ls -a")
	if hiddenOut.text != longName+"  0\r\n" {
		return false
	}
	if err = v.Mkdir("/notes"); err != nil {
		return false
	}
	data := make([]byte, 2049)
	for i := range data {
		data[i] = byte(i * 37)
	}
	if err = v.WriteFile("/notes/data.bin", data); err != nil {
		return false
	}
	f, err := v.Open("/NOTES/DATA.BIN")
	if err != nil {
		return false
	}
	read := make([]byte, len(data))
	n, err := f.Read(read)
	if err != nil || n != len(data) {
		return false
	}
	for i := range read {
		if read[i] != data[i] {
			return false
		}
	}
	list, err := v.ReadDir("/notes")
	if err != nil || len(list) != 1 {
		return false
	}
	if !fat32.Is(v.Remove("/notes"), fat32.ErrNotEmpty) {
		return false
	}
	o := &output{}
	sh := shell.New(v, o)
	sh.Execute("cd notes")
	sh.Execute("write hi.txt 'hello card'")
	sh.Execute("cat hi.txt")
	if sh.Cwd != "/notes" || o.text != "hello card\r\n" {
		return false
	}
	sh.Execute("cd ../..")
	if sh.Cwd != "/" {
		return false
	}
	if err = v.Remove("/notes/data.bin"); err != nil {
		return false
	}
	if err = v.Remove("/notes/hi.txt"); err != nil {
		return false
	}
	if err = v.Remove("/notes"); err != nil {
		return false
	}
	return v.Sync() == nil
}
func main() {
	if !run() {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
