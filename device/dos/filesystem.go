package dos

const (
	OpenRead      = 0
	OpenWrite     = 1
	OpenReadWrite = 2
)

// File is an MS-DOS file handle. It is intentionally small for programs that
// cannot afford the complete portable os package in a 16-bit segment.
type File struct {
	Handle uint16
}

func OpenFile(path string, mode byte) (*File, error) {
	name := cString(path)
	regs := Registers{AX: 0x3d00 | uint16(mode&3), DX: nearPointer(name)}
	interrupt21(&regs)
	if err := result(&regs); err != nil {
		return nil, err
	}
	return &File{Handle: regs.AX}, nil
}

func CreateFile(path string, attributes uint16) (*File, error) {
	name := cString(path)
	regs := Registers{AX: 0x3c00, CX: attributes, DX: nearPointer(name)}
	interrupt21(&regs)
	if err := result(&regs); err != nil {
		return nil, err
	}
	return &File{Handle: regs.AX}, nil
}

func (f *File) Close() error {
	if f == nil {
		return Error(6)
	}
	regs := Registers{AX: 0x3e00, BX: f.Handle}
	interrupt21(&regs)
	return result(&regs)
}

func (f *File) Read(data []byte) (int, error) {
	if f == nil {
		return 0, Error(6)
	}
	regs := Registers{AX: 0x3f00, BX: f.Handle, CX: uint16(len(data)), DX: nearPointer(data)}
	interrupt21(&regs)
	return int(regs.AX), result(&regs)
}

func (f *File) Write(data []byte) (int, error) {
	if f == nil {
		return 0, Error(6)
	}
	regs := Registers{AX: 0x4000, BX: f.Handle, CX: uint16(len(data)), DX: nearPointer(data)}
	interrupt21(&regs)
	return int(regs.AX), result(&regs)
}

func (f *File) WriteAll(data []byte) error {
	for len(data) > 0 {
		written, err := f.Write(data)
		if err != nil {
			return err
		}
		if written <= 0 || written > len(data) {
			return Error(5)
		}
		data = data[written:]
	}
	return nil
}
