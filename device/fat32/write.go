package fat32

import "renvo.dev/std/strings"

// Sync waits for the block device's writes to complete. Metadata is write-through.
func (v *Volume) Sync() error {
	err := v.dev.Sync()
	if err != nil {
		v.failed = true
	}
	return err
}

func (v *Volume) write(lba uint32, b []byte) error {
	if v.failed {
		return ErrReadOnly
	}
	v.dirValid = false
	v.fatValid = false
	if lba < v.base || lba >= v.base+v.total || uint32(len(b)/512) > v.base+v.total-lba {
		return ErrCorrupt
	}
	err := v.dev.WriteBlocks(lba, b)
	if err != nil {
		v.failed = true
	}
	return err
}

// Set all enabled FAT copies, preserving reserved high bits. A partial metadata
// failure poisons this mount: callers must repair/remount before further writes.
func (v *Volume) setFAT(c, n uint32) error {
	if c > v.clusters+1 {
		return ErrCorrupt
	}
	first, last := v.active, v.active+1
	if v.mirror {
		first = 0
		last = v.copies
	}
	var sector [512]byte
	for table := first; table < last; table++ {
		lba := v.fat + table*v.fatSize + c/128
		if err := v.dev.ReadBlocks(lba, sector[:]); err != nil {
			return err
		}
		offset := int(c%128) * 4
		put32(sector[:], offset, u32(sector[:], offset)&0xf0000000|n&0x0fffffff)
		if err := v.write(lba, sector[:]); err != nil {
			return err
		}
	}
	return nil
}

func (v *Volume) beginWrite() error {
	if v.failed {
		return ErrReadOnly
	}
	// FSInfo counts are only hints. Mark them unknown before allocation rather
	// than leaving stale free counts on either the primary or backup boot area.
	if v.fsinfo != 0 {
		var b [512]byte
		for i := 0; i < 2; i++ {
			if i == 1 && v.backup == 0 {
				break
			}
			lba := v.base + v.fsinfo
			if i == 1 {
				lba += v.backup
			}
			if err := v.dev.ReadBlocks(lba, b[:]); err != nil {
				return err
			}
			if u32(b[:], 0) == 0x41615252 && u32(b[:], 484) == 0x61417272 && signature(b[:]) {
				put32(b[:], 488, 0xffffffff)
				put32(b[:], 492, 0xffffffff)
				if err := v.write(lba, b[:]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (v *Volume) allocate() (uint32, error) {
	for scanned := uint32(0); scanned < v.clusters; scanned++ {
		c := v.nextFree
		v.nextFree++
		if v.nextFree >= v.clusters+2 {
			v.nextFree = 2
		}
		b, err := v.fatSector(c)
		if err != nil {
			return 0, err
		}
		if u32(b, int(c%128)*4)&0x0fffffff != 0 {
			continue
		}
		if err = v.setFAT(c, 0x0fffffff); err != nil {
			return 0, err
		}
		// Clear before publishing: new directory slots and file tails must never
		// expose previous owners' bytes. Reuse one sector of zeroed scratch.
		var zero [512]byte
		for n := uint32(0); n < v.spc; n++ {
			if err = v.write(v.sector(c)+n, zero[:]); err != nil {
				return 0, err
			}
		}
		return c, nil
	}
	return 0, ErrFull
}

func (v *Volume) release(c uint32) error {
	// Validate before freeing: a loop must not cause a partially freed chain.
	var chain chainGuard
	for n := c; n != 0; {
		if !chain.accept(n) {
			return ErrCorrupt
		}
		next, err := v.next(n)
		if err != nil {
			return err
		}
		n = next
	}
	for hops := uint32(0); c != 0; hops++ {
		if hops >= v.clusters {
			return ErrCorrupt
		}
		n, err := v.next(c)
		if err != nil {
			return err
		}
		if err = v.setFAT(c, 0); err != nil {
			return err
		}
		c = n
	}
	return nil
}

func (v *Volume) parent(path string) (Entry, string, error) {
	path = Clean("/", path)
	if path == "/" {
		return Entry{}, "", ErrName
	}
	i := len(path) - 1
	for i > 0 && path[i] != '/' {
		i--
	}
	p, err := v.Stat(path[:i])
	if err != nil {
		return Entry{}, "", err
	}
	if !p.IsDir() {
		return Entry{}, "", ErrNotDir
	}
	return p, path[i+1:], nil
}

func encodeName(name string) ([11]byte, error) {
	var out [11]byte
	for i := range out {
		out[i] = ' '
	}
	parts := strings.Split(name, ".")
	if len(parts) > 2 || len(parts[0]) < 1 || len(parts[0]) > 8 {
		return out, ErrName
	}
	if len(parts) == 2 && (len(parts[1]) < 1 || len(parts[1]) > 3) {
		return out, ErrName
	}
	for part, s := range parts {
		base := 0
		if part == 1 {
			base = 8
		}
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'a' && c <= 'z' {
				c -= 32
			}
			if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.Contains("_$~!#%&-{}()@'`", s[i:i+1])) {
				return out, ErrName
			}
			out[base+i] = c
		}
	}
	return out, nil
}

func (v *Volume) emptySlot(c uint32) (uint32, int, error) {
	var chain chainGuard
	for hops := uint32(0); hops < v.clusters; hops++ {
		if !v.validCluster(c) || !chain.accept(c) {
			return 0, 0, ErrCorrupt
		}
		for n := uint32(0); n < v.spc; n++ {
			lba := v.sector(c) + n
			b, err := v.directorySector(lba)
			if err != nil {
				return 0, 0, err
			}
			for i := 0; i < 512; i += 32 {
				if b[i] == 0 || b[i] == 0xe5 {
					return lba, i, nil
				}
			}
		}
		n, err := v.next(c)
		if err != nil {
			return 0, 0, err
		}
		if n == 0 {
			n, err = v.allocate()
			if err != nil {
				return 0, 0, err
			}
			if err = v.setFAT(c, n); err != nil {
				return 0, 0, err
			}
			return v.sector(n), 0, nil
		}
		c = n
	}
	return 0, 0, ErrCorrupt
}

func (v *Volume) publish(e Entry, name [11]byte, fresh bool) error {
	var b [512]byte
	if err := v.dev.ReadBlocks(e.lba, b[:]); err != nil {
		return err
	}
	r := b[e.offset : e.offset+32]
	if fresh {
		for i := range r {
			r[i] = 0
		}
		copy(r[:11], name[:])
		put16(r, 16, 33)
		put16(r, 18, 33)
		put16(r, 24, 33)
	}
	r[11] = e.Attributes
	put16(r, 20, e.cluster>>16)
	put16(r, 26, e.cluster)
	put32(r, 28, e.Size)
	if err := v.write(e.lba, b[:]); err != nil {
		return err
	}
	return v.Sync()
}

// WriteFile creates or replaces a regular file. Replacements are copy-on-write:
// keep the old chain until the new data and directory entry have been synced.
// New names use portable ASCII 8.3; existing VFAT long names may be overwritten.
// FAT is not a journal: power failure can still leak clusters or tear a sector.
func (v *Volume) WriteFile(path string, data []byte) error {
	parent, name, err := v.parent(path)
	if err != nil {
		return err
	}
	e, err := v.Stat(path)
	fresh := Is(err, ErrNotFound)
	var short [11]byte
	if !fresh && err != nil {
		return err
	}
	if fresh {
		short, err = encodeName(name)
		if err != nil {
			return err
		}
		e.Attributes = 32
	} else {
		if e.IsDir() {
			return ErrIsDir
		}
		if e.Attributes&1 != 0 {
			return ErrReadOnly
		}
	}
	if err = v.beginWrite(); err != nil {
		return err
	}
	if fresh {
		e.lba, e.offset, err = v.emptySlot(parent.cluster)
		if err != nil {
			return err
		}
	}
	old := e.cluster
	first, last := uint32(0), uint32(0)
	size := len(data)
	for len(data) > 0 {
		c, allocErr := v.allocate()
		if allocErr != nil {
			if !v.failed {
				v.release(first)
			}
			return allocErr
		}
		if first == 0 {
			first = c
		} else {
			if err = v.setFAT(last, c); err != nil {
				return err
			}
		}
		last = c
		n := len(data)
		if n > int(v.spc*512) {
			n = int(v.spc * 512)
		}
		full := n / 512 * 512
		if full > 0 {
			if err = v.write(v.sector(c), data[:full]); err != nil {
				return err
			}
		}
		if full < n {
			var tail [512]byte
			copy(tail[:], data[full:n])
			if err = v.write(v.sector(c)+uint32(full/512), tail[:]); err != nil {
				return err
			}
		}
		data = data[n:]
	}
	if err = v.Sync(); err != nil {
		v.failed = true
		return err
	}
	e.cluster = first
	e.Size = uint32(size)
	if err = v.publish(e, short, fresh); err != nil {
		v.failed = true
		return err
	}
	if old != 0 {
		if err = v.release(old); err != nil {
			return err
		}
	}
	return v.Sync()
}

// Mkdir creates one directory with '.' and '..'; it does not create parents.
func (v *Volume) Mkdir(path string) error {
	p, name, err := v.parent(path)
	if err != nil {
		return err
	}
	if _, err = v.Stat(path); err == nil {
		return ErrExists
	} else if !Is(err, ErrNotFound) {
		return err
	}
	short, err := encodeName(name)
	if err != nil {
		return err
	}
	if err = v.beginWrite(); err != nil {
		return err
	}
	lba, offset, err := v.emptySlot(p.cluster)
	if err != nil {
		return err
	}
	c, err := v.allocate()
	if err != nil {
		return err
	}
	var b [512]byte
	for i := 0; i < 2; i++ {
		r := b[i*32 : i*32+32]
		for n := 0; n < 11; n++ {
			r[n] = ' '
		}
		r[0] = '.'
		r[11] = 16
		cluster := c
		if i == 1 {
			r[1] = '.'
			cluster = p.cluster
			if cluster == v.root {
				cluster = 0
			}
		}
		put16(r, 20, cluster>>16)
		put16(r, 26, cluster)
		put16(r, 24, 33)
	}
	if err = v.write(v.sector(c), b[:]); err != nil {
		return err
	}
	if err = v.Sync(); err != nil {
		v.failed = true
		return err
	}
	return v.publish(Entry{Attributes: 16, cluster: c, lba: lba, offset: offset}, short, true)
}

// Remove unlinks a file or empty directory, then releases its chain. It never
// recursively deletes. Valid associated LFN slots are unlinked as well.
func (v *Volume) Remove(path string) error {
	if Clean("/", path) == "/" {
		return ErrReadOnly
	}
	e, err := v.Stat(path)
	if err != nil {
		return err
	}
	if e.Attributes&1 != 0 {
		return ErrReadOnly
	}
	if e.IsDir() {
		items, err := v.readDirectory(e.cluster)
		if err != nil {
			return err
		}
		if len(items) != 0 {
			return ErrNotEmpty
		}
	}
	if err = v.beginWrite(); err != nil {
		return err
	}
	var b [512]byte
	if err = v.dev.ReadBlocks(e.lba, b[:]); err != nil {
		return err
	}
	b[e.offset] = 0xe5
	if err = v.write(e.lba, b[:]); err != nil {
		return err
	}
	for _, slot := range e.longSlots {
		if err = v.dev.ReadBlocks(slot.lba, b[:]); err != nil {
			return err
		}
		b[slot.offset] = 0xe5
		if err = v.write(slot.lba, b[:]); err != nil {
			return err
		}
	}
	if err = v.Sync(); err != nil {
		v.failed = true
		return err
	}
	if e.cluster != 0 {
		if err = v.release(e.cluster); err != nil {
			return err
		}
	}
	return v.Sync()
}
