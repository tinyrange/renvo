// Package fat32 implements a synchronous FAT32 volume over 512-byte blocks.
// A Volume is single-owner. Mount is read-only until a mutating method is called.
// FAT is not journaled: keep power connected through Sync, and back up media.
package fat32

import (
	"renvo.dev/device/block"
	"renvo.dev/std/io"
	"renvo.dev/std/strings"
	"renvo.dev/std/unicode/utf8"
)

type Error string

func (e Error) Error() string { return string(e) }

// Is matches a FAT error code without treating an unrelated device error with
// the same text as a match. Device errors pass through the filesystem unchanged.
func Is(err error, target Error) bool { code, ok := err.(Error); return ok && code == target }

const (
	ErrFormat   Error = "fat32: invalid or unsupported volume"
	ErrCorrupt  Error = "fat32: corrupt cluster chain"
	ErrNotFound Error = "fat32: path not found"
	ErrNotDir   Error = "fat32: not a directory"
	ErrIsDir    Error = "fat32: is a directory"
	ErrExists   Error = "fat32: already exists"
	ErrName     Error = "fat32: new names must be ASCII 8.3"
	ErrFull     Error = "fat32: no free space"
	ErrReadOnly Error = "fat32: read-only entry or failed volume"
	ErrNotEmpty Error = "fat32: directory not empty"
)

// Volume caches one FAT sector and one directory sector. File data bypasses
// metadata caches, batching physically contiguous sectors within each cluster.
type Volume struct {
	dev                                                                  block.Device
	base, total, fat, fatSize, data, root, clusters, spc, copies, active uint32
	mirror                                                               bool
	fsinfo, backup                                                       uint32
	fatCache, dirCache                                                   [512]byte
	fatLBA, dirLBA                                                       uint32
	fatValid, dirValid, failed                                           bool
	nextFree                                                             uint32
}

func u16(b []byte, p int) uint32      { return uint32(b[p]) | uint32(b[p+1])<<8 }
func u32(b []byte, p int) uint32      { return u16(b, p) | u16(b, p+2)<<16 }
func put16(b []byte, p int, n uint32) { b[p] = byte(n); b[p+1] = byte(n >> 8) }
func put32(b []byte, p int, n uint32) { put16(b, p, n); put16(b, p+2, n>>16) }
func signature(b []byte) bool         { return b[510] == 0x55 && b[511] == 0xaa }
func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
func trimSpace(s string) string {
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}

// Mount accepts an unpartitioned FAT32 volume or the first FAT32 primary MBR
// partition (0x0b/0x0c). FAT12/16, exFAT, GPT and extended partitions are rejected.
func Mount(dev block.Device) (*Volume, error) {
	if dev == nil || dev.Blocks() == 0 {
		return nil, ErrFormat
	}
	v := &Volume{dev: dev, nextFree: 2}
	b := v.dirCache[:]
	if err := dev.ReadBlocks(0, b); err != nil {
		return nil, err
	}
	if !signature(b) {
		return nil, ErrFormat
	}
	limit := dev.Blocks()
	if u16(b, 11) != 512 || b[13] == 0 || u16(b, 14) == 0 {
		found := false
		for i := 0; i < 4; i++ {
			p := 446 + i*16
			if b[p+4] != 11 && b[p+4] != 12 {
				continue
			}
			v.base = u32(b, p+8)
			limit = u32(b, p+12)
			if v.base == 0 || limit == 0 || v.base >= dev.Blocks() || limit > dev.Blocks()-v.base {
				return nil, ErrFormat
			}
			found = true
			break
		}
		if !found {
			return nil, ErrFormat
		}
		if err := dev.ReadBlocks(v.base, b); err != nil {
			return nil, err
		}
	}
	v.total = u32(b, 32)
	v.spc = uint32(b[13])
	v.copies = uint32(b[16])
	v.fatSize = u32(b, 36)
	reserved := u16(b, 14)
	if !signature(b) || u16(b, 11) != 512 || v.spc == 0 || v.spc > 128 || v.spc&(v.spc-1) != 0 || reserved == 0 || v.copies < 1 || v.copies > 2 || u16(b, 17) != 0 || u16(b, 19) != 0 || u16(b, 22) != 0 || u16(b, 42) != 0 || v.total == 0 || v.total > limit || v.fatSize == 0 || v.fatSize > (v.total-reserved)/v.copies || reserved >= v.total {
		return nil, ErrFormat
	}
	overhead := reserved + v.copies*v.fatSize
	if overhead >= v.total {
		return nil, ErrFormat
	}
	v.clusters = (v.total - overhead) / v.spc
	if v.clusters < 65525 || v.clusters >= 0x0ffffff5 || v.fatSize < (v.clusters+2+127)/128 {
		return nil, ErrFormat
	}
	v.root = u32(b, 44) & 0x0fffffff
	if !v.validCluster(v.root) {
		return nil, ErrFormat
	}
	flags := u16(b, 40)
	v.mirror = flags&128 == 0
	if !v.mirror {
		v.active = flags & 15
		if v.active >= v.copies {
			return nil, ErrFormat
		}
	}
	v.fat = v.base + reserved
	v.data = v.base + overhead
	v.fsinfo = u16(b, 48)
	v.backup = u16(b, 50)
	if v.fsinfo == 0 || v.fsinfo >= reserved {
		v.fsinfo = 0
	}
	if v.backup == 0 || v.backup >= reserved || v.fsinfo >= reserved-v.backup {
		v.backup = 0
	}
	return v, nil
}

func (v *Volume) validCluster(c uint32) bool { return c >= 2 && c < v.clusters+2 }
func (v *Volume) sector(c uint32) uint32     { return v.data + (c-2)*v.spc }
func (v *Volume) fatSector(c uint32) ([]byte, error) {
	lba := v.fat + v.active*v.fatSize + c/128
	if !v.fatValid || v.fatLBA != lba {
		v.fatValid = false
		if err := v.dev.ReadBlocks(lba, v.fatCache[:]); err != nil {
			return nil, err
		}
		v.fatLBA = lba
		v.fatValid = true
	}
	return v.fatCache[:], nil
}
func (v *Volume) next(c uint32) (uint32, error) {
	if !v.validCluster(c) {
		return 0, ErrCorrupt
	}
	b, err := v.fatSector(c)
	if err != nil {
		return 0, err
	}
	n := u32(b, int(c%128)*4) & 0x0fffffff
	if n >= 0x0ffffff8 {
		return 0, nil
	}
	if !v.validCluster(n) {
		return 0, ErrCorrupt
	}
	return n, nil
}
func (v *Volume) directorySector(lba uint32) ([]byte, error) {
	if !v.dirValid || v.dirLBA != lba {
		v.dirValid = false
		if err := v.dev.ReadBlocks(lba, v.dirCache[:]); err != nil {
			return nil, err
		}
		v.dirLBA = lba
		v.dirValid = true
	}
	return v.dirCache[:], nil
}

// Entry names prefer validated VFAT long names; ShortName remains usable as an
// alias. Size is bytes; directories do not carry meaningful sizes.
type Entry struct {
	Name, ShortName string
	Size            uint32
	Attributes      byte
	cluster, lba    uint32
	offset          int
	longSlots       []directorySlot
}

type directorySlot struct {
	lba    uint32
	offset int
}

func (e Entry) IsDir() bool { return e.Attributes&16 != 0 }

// IsHidden recognizes both FAT's hidden attribute and Unix-style dot names.
// ReadDir still returns them: visibility is a shell/UI policy, not lookup policy.
func (e Entry) IsHidden() bool { return e.Attributes&2 != 0 || strings.HasPrefix(e.Name, ".") }
func (v *Volume) rootEntry() Entry {
	return Entry{Name: "/", ShortName: "/", Attributes: 16, cluster: v.root}
}

// Clean resolves slash-separated paths, '.' and '..' without escaping root.
func Clean(cwd, path string) string {
	if path == "" {
		return cwd
	}
	if path[0] != '/' {
		path = cwd + "/" + path
	}
	parts := strings.Split(path, "/")
	out := []string{}
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		if p == ".." {
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		} else {
			out = append(out, p)
		}
	}
	return "/" + strings.Join(out, "/")
}
func (v *Volume) Stat(path string) (Entry, error) {
	path = Clean("/", path)
	e := v.rootEntry()
	for _, name := range strings.Split(path, "/") {
		if name == "" {
			continue
		}
		if !e.IsDir() {
			return Entry{}, ErrNotDir
		}
		list, err := v.readDirectory(e.cluster)
		if err != nil {
			return Entry{}, err
		}
		found := false
		for _, item := range list {
			if lower(name) == lower(item.Name) || lower(name) == lower(item.ShortName) {
				e = item
				found = true
				break
			}
		}
		if !found {
			return Entry{}, ErrNotFound
		}
	}
	return e, nil
}
func (v *Volume) ReadDir(path string) ([]Entry, error) {
	e, err := v.Stat(path)
	if err != nil {
		return nil, err
	}
	if !e.IsDir() {
		return nil, ErrNotDir
	}
	return v.readDirectory(e.cluster)
}

func shortName(b []byte) string {
	base := trimSpace(string(b[:8]))
	ext := trimSpace(string(b[8:11]))
	if b[12]&8 != 0 {
		base = lower(base)
	}
	if b[12]&16 != 0 {
		ext = lower(ext)
	}
	if ext != "" {
		base += "." + ext
	}
	return base
}
func checksum(b []byte) byte {
	var n byte
	for i := 0; i < 11; i++ {
		n = (n >> 1) | (n << 7)
		n += b[i]
	}
	return n
}

// chainGuard uses exponentially spaced checkpoints to detect cycles without
// allocating a visited-cluster set or issuing additional FAT reads.
type chainGuard struct{ checkpoint, span, used uint32 }

func (g *chainGuard) accept(c uint32) bool {
	if g.span == 0 {
		g.checkpoint = c
		g.span = 1
		return true
	}
	if c == g.checkpoint {
		return false
	}
	g.used++
	if g.used == g.span {
		g.checkpoint = c
		g.span *= 2
		g.used = 0
	}
	return true
}

func (v *Volume) readDirectory(c uint32) ([]Entry, error) {
	entries := []Entry{}
	var chain chainGuard
	var name [260]uint16
	seq := 0
	nameUnits := 0
	var sum byte
	longValid := false
	longSlots := []directorySlot{}
	for hops := uint32(0); c != 0; hops++ {
		if hops >= v.clusters || !v.validCluster(c) || !chain.accept(c) {
			return nil, ErrCorrupt
		}
		for sector := uint32(0); sector < v.spc; sector++ {
			lba := v.sector(c) + sector
			b, err := v.directorySector(lba)
			if err != nil {
				return nil, err
			}
			for offset := 0; offset < 512; offset += 32 {
				r := b[offset : offset+32]
				if r[0] == 0 {
					return entries, nil
				}
				if r[0] == 0xe5 {
					seq = 0
					longValid = false
					continue
				}
				if r[11] == 15 {
					order := int(r[0] & 31)
					if r[0]&64 != 0 {
						longValid = true
						longSlots = nil
						seq = order
						nameUnits = order * 13
						sum = r[13]
						for i := range name {
							name[i] = 0xffff
						}
					}
					if !longValid || r[0]&0xa0 != 0 || order < 1 || order > 20 || order != seq || r[13] != sum || r[12] != 0 || u16(r, 26) != 0 {
						seq = 0
						longValid = false
						continue
					}
					positions := [13]int{1, 3, 5, 7, 9, 14, 16, 18, 20, 22, 24, 28, 30}
					for i, p := range positions {
						name[(order-1)*13+i] = uint16(u16(r, p))
					}
					seq--
					longSlots = append(longSlots, directorySlot{lba, offset})
					continue
				}
				if r[11]&8 != 0 {
					seq = 0
					longValid = false
					continue
				}
				e := Entry{ShortName: shortName(r), Size: u32(r, 28), Attributes: r[11], cluster: (u16(r, 20)<<16 | u16(r, 26)) & 0x0fffffff, lba: lba, offset: offset}
				e.Name = e.ShortName
				if longValid && seq == 0 && checksum(r) == sum && name[0] != 0 && name[0] != 0xffff {
					e.longSlots = longSlots
					text := ""
					// Exact multiples of 13 need no terminator on disk. Never
					// decode beyond the slots belonging to this LFN sequence.
					for i := 0; i < nameUnits && i < 255 && name[i] != 0 && name[i] != 0xffff; i++ {
						ch := rune(name[i])
						if ch >= 0xd800 && ch <= 0xdbff && i+1 < nameUnits && i+1 < 255 && name[i+1] >= 0xdc00 && name[i+1] <= 0xdfff {
							ch = 0x10000 + (ch-0xd800)*1024 + rune(name[i+1]) - 0xdc00
							i++
						}
						var encoded [4]byte
						n := utf8.EncodeRune(encoded[:], ch)
						text += string(encoded[:n])
					}
					if text != "" {
						e.Name = text
					}
				}
				seq = 0
				name[0] = 0
				longValid = false
				if e.Name != "." && e.Name != ".." {
					entries = append(entries, e)
				}
			}
		}
		var err error
		c, err = v.next(c)
		if err != nil {
			return nil, err
		}
	}
	return entries, nil
}

// File is a streaming, forward-only reader; opening does not load its contents.
// Do not mutate the volume while a File is open.
type File struct {
	chain                           chainGuard
	v                               *Volume
	entry                           Entry
	position, cluster, within, hops uint32
	scratch                         [512]byte
}

func (f *File) Size() uint32 { return f.entry.Size }

func (v *Volume) Open(path string) (*File, error) {
	e, err := v.Stat(path)
	if err != nil {
		return nil, err
	}
	if e.IsDir() {
		return nil, ErrIsDir
	}
	if e.Size > 0 && !v.validCluster(e.cluster) {
		return nil, ErrCorrupt
	}
	f := &File{v: v, entry: e, cluster: e.cluster}
	f.chain.accept(e.cluster)
	return f, nil
}
func (f *File) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if f.position == f.entry.Size {
		return 0, io.EOF
	}
	done := 0
	v := f.v
	for len(p) > 0 && f.position < f.entry.Size {
		if f.within == v.spc*512 {
			n, err := v.next(f.cluster)
			if err != nil {
				return done, err
			}
			f.hops++
			if n == 0 || f.hops >= v.clusters || !f.chain.accept(n) {
				return done, ErrCorrupt
			}
			f.cluster = n
			f.within = 0
		}
		n := len(p)
		remaining := f.entry.Size - f.position
		if uint32(n) > remaining {
			n = int(remaining)
		}
		available := v.spc*512 - f.within
		if uint32(n) > available {
			n = int(available)
		}
		lba := v.sector(f.cluster) + f.within/512
		if f.within%512 == 0 && n >= 512 {
			n = n / 512 * 512
			if err := v.dev.ReadBlocks(lba, p[:n]); err != nil {
				return done, err
			}
		} else {
			offset := int(f.within % 512)
			if n > 512-offset {
				n = 512 - offset
			}
			if err := v.dev.ReadBlocks(lba, f.scratch[:]); err != nil {
				return done, err
			}
			copy(p[:n], f.scratch[offset:offset+n])
		}
		f.within += uint32(n)
		f.position += uint32(n)
		done += n
		p = p[n:]
	}
	return done, nil
}
