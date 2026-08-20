package bufio

import (
	"errors"
	"io"
	"unicode/utf8"
)

const MaxScanTokenSize = 64 * 1024
const startBufSize = 4096

type SplitFunc func(data []byte, atEOF bool) (advance int, token []byte, err error)

var ErrInvalidUnreadByte = errors.New("bufio: invalid use of UnreadByte")
var ErrInvalidUnreadRune = errors.New("bufio: invalid use of UnreadRune")
var ErrBufferFull = errors.New("bufio: buffer full")
var ErrNegativeCount = errors.New("bufio: negative count")
var ErrTooLong = errors.New("bufio.Scanner: token too long")
var ErrBadReadCount = errors.New("bufio.Scanner: Read returned impossible count")
var ErrFinalToken = errors.New("final token")

type Scanner struct {
	r            io.Reader
	split        SplitFunc
	maxTokenSize int
	buf          []byte
	start        int
	end          int
	token        []byte
	err          error
	done         bool
	scanCalled   bool
	emptyTokens  int
}

func NewScanner(r io.Reader) *Scanner {
	scanner := &Scanner{r: r, maxTokenSize: MaxScanTokenSize}
	scanner.split = ScanLines
	return scanner
}

func (s *Scanner) Scan() bool {
	if s.done {
		return false
	}
	s.scanCalled = true
	for {
		if s.end > s.start || s.err != nil {
			advance, token, err := s.split(s.buf[s.start:s.end], s.err != nil)
			if err != nil {
				if err == ErrFinalToken {
					s.token = token
					s.done = true
					return token != nil
				}
				s.err = err
				s.done = true
				return false
			}
			if advance < 0 || advance > s.end-s.start {
				s.err = errors.New("bufio.Scanner: SplitFunc returns invalid advance count")
				s.done = true
				return false
			}
			s.start += advance
			if token != nil {
				s.token = token
				if advance == 0 && s.err != nil {
					s.emptyTokens++
					if s.emptyTokens > 100 {
						s.done = true
						return false
					}
				} else {
					s.emptyTokens = 0
				}
				return true
			}
		}
		if s.err != nil {
			s.done = true
			return false
		}
		if s.start > 0 {
			copy(s.buf, s.buf[s.start:s.end])
			s.end -= s.start
			s.start = 0
		}
		if s.buf == nil {
			size := startBufSize
			if size > s.maxTokenSize {
				size = s.maxTokenSize
			}
			s.buf = make([]byte, size)
		} else if s.end == len(s.buf) {
			if len(s.buf) >= s.maxTokenSize {
				s.err = ErrTooLong
				s.done = true
				return false
			}
			newSize := len(s.buf) * 2
			if newSize > s.maxTokenSize {
				newSize = s.maxTokenSize
			}
			newBuf := make([]byte, newSize)
			copy(newBuf, s.buf[:s.end])
			s.buf = newBuf
		}
		n, err := s.r.Read(s.buf[s.end:])
		if n < 0 || s.end+n > len(s.buf) {
			s.err = ErrBadReadCount
			continue
		}
		s.end += n
		if err != nil {
			s.err = err
		}
		if n == 0 && err == nil {
			s.err = io.EOF
		}
	}
}

func (s *Scanner) Bytes() []byte { return s.token }
func (s *Scanner) Text() string  { return string(s.token) }
func (s *Scanner) Err() error {
	if s.err == io.EOF {
		return nil
	}
	return s.err
}
func (s *Scanner) Split(split SplitFunc) {
	if s.scanCalled {
		panic("Split called after Scan")
	}
	s.split = split
}
func (s *Scanner) Buffer(buf []byte, max int) {
	if s.scanCalled {
		panic("Buffer called after Scan")
	}
	s.buf = buf[:cap(buf)]
	s.maxTokenSize = max
}

func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '' {
		return data[:len(data)-1]
	}
	return data
}

func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i, b := range data {
		if b == 10 {
			return i + 1, dropCR(data[:i]), nil
		}
	}
	if atEOF && len(data) != 0 {
		return len(data), dropCR(data), nil
	}
	return 0, nil, nil
}

func ScanBytes(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if len(data) == 0 {
		return 0, nil, nil
	}
	return 1, data[:1], nil
}

func ScanWords(data []byte, atEOF bool) (advance int, token []byte, err error) {
	start := 0
	for start < len(data) && isSpace(data[start]) {
		start++
	}
	for i := start; i < len(data); i++ {
		if isSpace(data[i]) {
			return i + 1, data[start:i], nil
		}
	}
	if atEOF && start < len(data) {
		return len(data), data[start:], nil
	}
	return start, nil, nil
}

func ScanRunes(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if len(data) == 0 {
		return 0, nil, nil
	}
	if data[0] < utf8.RuneSelf {
		return 1, data[:1], nil
	}
	_, width := utf8.DecodeRuneInString(string(data))
	if width == 1 && !atEOF && runeWidth(data[0]) > len(data) {
		return 0, nil, nil
	}
	return width, data[:width], nil
}

func runeWidth(b byte) int {
	if b < 0xc0 {
		return 1
	}
	if b < 0xe0 {
		return 2
	}
	if b < 0xf0 {
		return 3
	}
	return 4
}

func isSpace(b byte) bool {
	return b == 32 || b == 9 || b == 10 || b == 13 || b == 11 || b == 12
}
