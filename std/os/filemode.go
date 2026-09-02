package os

// FileMode represents a file's mode and permission bits. Its values match
// Go's os.FileMode so modes remain portable between host tools and Renvo
// targets.
type FileMode uint32

const ModeDir FileMode = 1 << 31
const ModeAppend FileMode = 1 << 30
const ModeExclusive FileMode = 1 << 29
const ModeTemporary FileMode = 1 << 28
const ModeSymlink FileMode = 1 << 27
const ModeDevice FileMode = 1 << 26
const ModeNamedPipe FileMode = 1 << 25
const ModeSocket FileMode = 1 << 24
const ModeSetuid FileMode = 1 << 23
const ModeSetgid FileMode = 1 << 22
const ModeCharDevice FileMode = 1 << 21
const ModeSticky FileMode = 1 << 20
const ModeIrregular FileMode = 1 << 19

// ModeType is the mask for file type bits. Regular files have no ModeType
// bits set.
const ModeType FileMode = ModeDir | ModeSymlink | ModeNamedPipe | ModeSocket | ModeDevice | ModeCharDevice | ModeIrregular

// ModePerm contains the portable Unix permission bits (rwxrwxrwx).
const ModePerm FileMode = 0o777

func (m FileMode) IsDir() bool { return m&ModeDir != 0 }

func (m FileMode) IsRegular() bool { return m&ModeType == 0 }

func (m FileMode) Perm() FileMode { return m & ModePerm }

func (m FileMode) Type() FileMode { return m & ModeType }

func (m FileMode) String() string {
	const letters = "dalTLDpSugct?"
	var buffer [32]byte
	length := 0
	for index := 0; index < len(letters); index++ {
		if m&(1<<uint(31-index)) != 0 {
			buffer[length] = letters[index]
			length++
		}
	}
	if length == 0 {
		buffer[0] = '-'
		length = 1
	}
	const permissions = "rwxrwxrwx"
	for index := 0; index < len(permissions); index++ {
		if m&(1<<uint(8-index)) != 0 {
			buffer[length] = permissions[index]
		} else {
			buffer[length] = '-'
		}
		length++
	}
	return string(buffer[:length])
}
