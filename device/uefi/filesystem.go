package uefi

import "unsafe"

var SimpleFileSystemProtocolGUID = GUID{0x11d26459964e5b22, 0x3b7269c9a000398e}
var LoadedImageProtocolGUID = GUID{0x11d295625b1b31a1, 0x3b7269c9a0003f8e}

const (
	FileModeRead   = uint64(1)
	FileModeWrite  = uint64(2)
	FileModeCreate = uint64(0x8000000000000000)
	FileReadOnly   = uint64(1)
	FileHidden     = uint64(2)
	FileSystem     = uint64(4)
	FileReserved   = uint64(8)
	FileDirectory  = uint64(16)
	FileArchive    = uint64(32)
)

type SimpleFileSystemProtocol struct {
	Revision   uint64
	OpenVolume uintptr
}

type File struct {
	Revision            uint64
	OpenFunction        uintptr
	CloseFunction       uintptr
	DeleteFunction      uintptr
	ReadFunction        uintptr
	WriteFunction       uintptr
	GetPositionFunction uintptr
	SetPositionFunction uintptr
	GetInfoFunction     uintptr
	SetInfoFunction     uintptr
	FlushFunction       uintptr
}

func OpenVolume() (*File, Status) {
	loadedImage, status := HandleProtocol(ImageHandle(), &LoadedImageProtocolGUID)
	if status.Failed() {
		return nil, status
	}
	device := Handle(loadWord(loadedImage, 24))
	value, status := HandleProtocol(device, &SimpleFileSystemProtocolGUID)
	if status.Failed() {
		return nil, status
	}
	filesystem := (*SimpleFileSystemProtocol)(unsafe.Pointer(value))
	var root *File
	status = Status(call2(filesystem.OpenVolume, pointer(unsafe.Pointer(filesystem)),
		pointer(unsafe.Pointer(&root))))
	return root, status
}

func (f *File) Open(path string, mode, attributes uint64) (*File, Status) {
	// Keep the firmware-visible name in this call frame. Besides avoiding an
	// allocation for the usual short UEFI paths, this guarantees that the
	// backing store remains live until the firmware Open call returns.
	var name [260]uint16
	encoded := encodeUTF16Z(name[:], path)
	if encoded == 0 {
		return nil, InvalidParameter
	}
	return f.OpenUTF16(name[:encoded], mode, attributes)
}
// OpenUTF16 opens a path already encoded as NUL-terminated UTF-16.
func (f *File) OpenUTF16(name []uint16, mode, attributes uint64) (*File, Status) {
	if f == nil {
		return nil, InvalidParameter
	}
	if len(name) == 0 || name[len(name)-1] != 0 {
		return nil, InvalidParameter
	}
	var child *File
	status := Status(call5(f.OpenFunction, pointer(unsafe.Pointer(f)),
		pointer(unsafe.Pointer(&child)), pointer(unsafe.Pointer(&name[0])),
		uintptr(mode), uintptr(attributes)))
	return child, status
}

func (f *File) Close() Status {
	if f == nil {
		return InvalidParameter
	}
	return Status(call1(f.CloseFunction, pointer(unsafe.Pointer(f))))
}

func (f *File) Delete() Status {
	if f == nil {
		return InvalidParameter
	}
	return Status(call1(f.DeleteFunction, pointer(unsafe.Pointer(f))))
}

func (f *File) Read(buffer []byte) (int, Status) {
	if f == nil {
		return 0, InvalidParameter
	}
	size := uintptr(len(buffer))
	address := uintptr(0)
	if len(buffer) != 0 {
		address = pointer(unsafe.Pointer(&buffer[0]))
	}
	status := Status(call3(f.ReadFunction, pointer(unsafe.Pointer(f)),
		pointer(unsafe.Pointer(&size)), address))
	return int(size), status
}

// ReadAddress reads directly into firmware-addressable memory. It is useful
// for large boot payloads which should not pass through Renvo's small arena.
func (f *File) ReadAddress(address, bytes uintptr) (uintptr, Status) {
	if f == nil || (address == 0 && bytes != 0) {
		return 0, InvalidParameter
	}
	size := bytes
	status := Status(call3(f.ReadFunction, pointer(unsafe.Pointer(f)),
		pointer(unsafe.Pointer(&size)), address))
	return size, status
}

func (f *File) Write(buffer []byte) (int, Status) {
	if f == nil {
		return 0, InvalidParameter
	}
	size := uintptr(len(buffer))
	address := uintptr(0)
	if len(buffer) != 0 {
		address = pointer(unsafe.Pointer(&buffer[0]))
	}
	status := Status(call3(f.WriteFunction, pointer(unsafe.Pointer(f)),
		pointer(unsafe.Pointer(&size)), address))
	return int(size), status
}

func (f *File) SetPosition(position uint64) Status {
	if f == nil {
		return InvalidParameter
	}
	return Status(call2(f.SetPositionFunction, pointer(unsafe.Pointer(f)), uintptr(position)))
}

func (f *File) Flush() Status {
	if f == nil {
		return InvalidParameter
	}
	return Status(call1(f.FlushFunction, pointer(unsafe.Pointer(f))))
}
