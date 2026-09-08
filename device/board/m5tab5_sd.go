//go:build m5tab5

package board

import (
	"renvo.dev/device/block"
	"renvo.dev/device/esp32p4"
	"unsafe"
)

const sdBase = uintptr(0x50083000)
const sdResponse = uint32(1<<6 | 1<<8)
const sdErrors = uint32(0xbfc2)

// SDCard owns SDMMC slot 0 and its DMA scratch buffers. Do not use concurrently
// or alongside another SDMMC host. Transfers batch up to 32 KiB, with 64-byte
// aligned PSRAM bounce storage so arbitrary caller buffers are cache-safe.
// This driver uses SDHC/SDXC sector addressing, 4-bit SDR at 20 MHz (no UHS
// voltage switch). Both command and data CRCs are checked by the peripheral.
type SDCard struct {
	clock                          esp32p4.SystemTimer
	count, rca                     uint32
	buffer, descriptors            []byte
	dataAddress, descriptorAddress uintptr
	ready                          bool
	LastCommand, LastStatus        uint32
}

func sdAligned(size int) ([]byte, uintptr) {
	b := make([]byte, size+63)
	a := uintptr(unsafe.Pointer(&b[0]))
	offset := int((64 - a%64) % 64)
	return b[offset : offset+size], a + uintptr(offset)
}

func (s *SDCard) waitClear(offset uintptr, mask uint32) error {
	start := s.clock.Ticks()
	for load32(sdBase+offset)&mask != 0 {
		if s.clock.Ticks()-start > 16000000 {
			return block.ErrTimeout
		}
	}
	return nil
}

func (s *SDCard) command(index, arg, flags uint32) error {
	if err := s.waitClear(0x2c, 1<<31); err != nil {
		return err
	}
	s.LastCommand = index
	store32(sdBase+0x44, 0xffffffff)
	store32(sdBase+0x28, arg)
	store32(sdBase+0x2c, index|flags|1<<29|1<<31)
	start := s.clock.Ticks()
	for {
		status := load32(sdBase + 0x44)
		s.LastStatus = status
		if status&sdErrors != 0 {
			return block.ErrIO
		}
		if status&4 != 0 {
			return nil
		}
		if s.clock.Ticks()-start > 16000000 {
			return block.ErrTimeout
		}
	}
}

func (s *SDCard) clockUpdate() error {
	store32(sdBase+0x2c, 1<<31|1<<21|1<<13)
	return s.waitClear(0x2c, 1<<31)
}

func (s *SDCard) setClock(div uint32) error {
	store32(sdBase+0x10, 0)
	if err := s.clockUpdate(); err != nil {
		return err
	}
	store32(sdBase+8, div)
	store32(sdBase+12, 0)
	if err := s.clockUpdate(); err != nil {
		return err
	}
	store32(sdBase+0x10, 1)
	return s.clockUpdate()
}

// OpenSD initializes the Tab5's dedicated GPIO39-44 IOMUX SD bus. It does not
// write sectors or format the card. Reuse the returned card until reboot.
func OpenSD() (*SDCard, error) {
	s := &SDCard{}
	s.buffer, s.dataAddress = sdAligned(32768)
	s.descriptors, s.descriptorAddress = sdAligned(256)
	update32(clockBase+0x18, 0, 1<<14)
	update32(0x5011104c, 0, 1<<28)
	update32(0x5011104c, 1<<28, 0)
	// PLL160M / 4 = 40 MHz module clock; card divider 50 gives 400 kHz.
	update32(clockBase+0x34, 3<<22, 1<<24)
	update32(clockBase+0x38, 0x3fffffff, 3<<9|1<<13|3<<17|1<<23|7<<27|1<<8)
	update32(clockBase+0x38, 1<<8, 0)
	for pin := 39; pin <= 44; pin++ {
		update32(ioMuxBase+4+uintptr(pin*4), 7<<12|1<<7|3<<10, 1<<9|1<<8|3<<10)
	}
	s.clock.DelayMilliseconds(2)
	store32(sdBase, 7)
	if err := s.waitClear(0, 7); err != nil {
		return nil, err
	}
	store32(sdBase+0x24, 0)
	store32(sdBase+0x14, 0xffffffff)
	store32(sdBase+0x18, 0)
	store32(sdBase+0x4c, 2<<28|7<<16|8)
	if err := s.setClock(50); err != nil {
		return nil, err
	}
	if err := s.command(0, 0, 1<<15); err != nil {
		return nil, err
	}
	if err := s.command(8, 0x1aa, sdResponse); err != nil {
		return nil, err
	}
	if load32(sdBase+0x30)&0xfff != 0x1aa {
		return nil, block.ErrUnsupported
	}
	for tries := 0; ; tries++ {
		if tries == 200 {
			return nil, block.ErrTimeout
		}
		if err := s.command(55, 0, sdResponse); err != nil {
			return nil, err
		}
		if err := s.command(41, 0x40ff8000, 1<<6); err != nil {
			return nil, err
		}
		ocr := load32(sdBase + 0x30)
		if ocr&(1<<31) != 0 {
			if ocr&(1<<30) == 0 {
				return nil, block.ErrUnsupported
			}
			break
		}
		s.clock.DelayMilliseconds(10)
	}
	if err := s.command(2, 0, sdResponse|1<<7); err != nil {
		return nil, err
	}
	if err := s.command(3, 0, sdResponse); err != nil {
		return nil, err
	}
	s.rca = load32(sdBase+0x30) & 0xffff0000
	if err := s.command(9, s.rca, sdResponse|1<<7); err != nil {
		return nil, err
	}
	// CSD v2: bits 69:48 encode C_SIZE, in 1024-sector units.
	if load32(sdBase+0x3c)>>30 != 1 {
		return nil, block.ErrUnsupported
	}
	cs := (load32(sdBase+0x38)&63)<<16 | load32(sdBase+0x34)>>16
	if cs == 0x3fffff {
		return nil, block.ErrUnsupported
	}
	s.count = (cs + 1) * 1024
	if err := s.command(7, s.rca, sdResponse); err != nil {
		return nil, err
	}
	if err := s.waitClear(0x48, 1<<9); err != nil {
		return nil, err
	}
	if err := s.command(55, s.rca, sdResponse); err != nil {
		return nil, err
	}
	if err := s.command(6, 2, sdResponse); err != nil {
		return nil, err
	}
	store32(sdBase+0x18, 1)
	if err := s.setClock(1); err != nil {
		return nil, err
	}
	s.ready = true
	return s, nil
}

func (s *SDCard) Blocks() uint32 { return s.count }
func (s *SDCard) Sync() error {
	if !s.ready {
		return block.ErrIO
	}
	if err := s.waitClear(0x48, 1<<9); err != nil {
		return err
	}
	if err := s.command(13, s.rca, sdResponse); err != nil {
		return err
	}
	if load32(sdBase+0x30)&0xfdffe008 != 0 {
		return block.ErrIO
	}
	return nil
}
func (s *SDCard) ReadBlocks(lba uint32, data []byte) error  { return s.transfer(lba, data, false) }
func (s *SDCard) WriteBlocks(lba uint32, data []byte) error { return s.transfer(lba, data, true) }

func (s *SDCard) transfer(lba uint32, data []byte, write bool) error {
	if !s.ready {
		return block.ErrIO
	}
	if !block.Valid(s.count, lba, len(data)) {
		return block.ErrRange
	}
	for len(data) > 0 {
		n := len(data)
		if n > len(s.buffer) {
			n = len(s.buffer)
		}
		if err := s.batch(lba, data[:n], write); err != nil {
			// A failed write is uncertain: do not retry or issue more writes.
			s.ready = false
			store32(sdBase+0x80, 0)
			store32(sdBase, 6)
			return err
		}
		lba += uint32(n / 512)
		data = data[n:]
	}
	return nil
}

func (s *SDCard) batch(lba uint32, data []byte, write bool) error {
	if err := s.Sync(); err != nil {
		return err
	}
	store32(sdBase, 6)
	if err := s.waitClear(0, 6); err != nil {
		return err
	}
	store32(sdBase+0x80, 1)
	if err := s.waitClear(0x80, 1); err != nil {
		return err
	}
	if write {
		copy(s.buffer, data)
	}
	cacheSync(s.dataAddress, len(data), 4)
	for i := range s.descriptors {
		s.descriptors[i] = 0
	}
	for offset, index := 0, 0; offset < len(data); index++ {
		size := len(data) - offset
		if size > 4096 {
			size = 4096
		}
		desc := s.descriptorAddress + uintptr(index*16)
		flags := uint32(1<<31 | 1<<4)
		if offset == 0 {
			flags |= 1 << 3
		}
		next := uint32(desc + 16)
		if offset+size == len(data) {
			flags |= 1 << 2
			next = 0
		}
		store32(desc, flags)
		store32(desc+4, uint32(size))
		store32(desc+8, uint32(s.dataAddress)+uint32(offset))
		store32(desc+12, next)
		offset += size
	}
	cacheSync(s.descriptorAddress, len(s.descriptors), 4)
	store32(sdBase+0x88, uint32(s.descriptorAddress))
	store32(sdBase+0x1c, 512)
	store32(sdBase+0x20, uint32(len(data)))
	store32(sdBase+0x8c, 0xffffffff)
	store32(sdBase, 1<<5|1<<25)
	store32(sdBase+0x80, 1<<7|2)
	store32(sdBase+0x90, 0x103)
	store32(sdBase+0x84, 1)
	cmd, flags := uint32(17), sdResponse|1<<9|1<<13
	if write {
		cmd = 24
		flags |= 1 << 10
	}
	if len(data) > 512 {
		cmd++
		flags |= 1 << 12
	}
	if err := s.command(cmd, lba, flags); err != nil {
		return err
	}
	if load32(sdBase+0x30)&0xfdffe008 != 0 {
		return block.ErrIO
	}
	start := s.clock.Ticks()
	dmaDone := uint32(2)
	if write {
		dmaDone = 1
	}
	for {
		status := load32(sdBase + 0x44)
		s.LastStatus = status
		if status&sdErrors != 0 || load32(sdBase+0x8c)&0x14 != 0 {
			return block.ErrIO
		}
		if status&8 != 0 && load32(sdBase+0x8c)&dmaDone != 0 && (len(data) == 512 || status&(1<<14) != 0) {
			break
		}
		if s.clock.Ticks()-start > 32000000 {
			return block.ErrTimeout
		}
	}
	if err := s.waitClear(0x48, 1<<9); err != nil {
		return err
	}
	store32(sdBase+0x80, 0)
	store32(sdBase, 0)
	if !write {
		cacheSync(s.dataAddress, len(data), 1)
		copy(data, s.buffer[:len(data)])
	}
	return s.Sync()
}
