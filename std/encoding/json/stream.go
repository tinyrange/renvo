package json

import (
	"errors"
	"io"
)

// Decoder reads successive JSON values from a stream. It may read ahead;
// unread bytes are retained for subsequent Decode calls.
type Decoder struct {
	reader  io.Reader
	buffer  []byte
	readErr error
	err     error
	strict  bool
	offset  int64
}

func NewDecoder(reader io.Reader) *Decoder { return &Decoder{reader: reader} }
func (d *Decoder) DisallowUnknownFields()  { d.strict = true }
func (d *Decoder) InputOffset() int64      { return d.offset }

func (d *Decoder) Decode(target any) error {
	if d.err != nil {
		return d.err
	}
	emptyReads := 0
	for {
		p := jsonParser{source: string(d.buffer)}
		p.space()
		start := p.at
		node := p.value(0)
		complete := p.err == nil && (node.kind != 'n' || p.at < len(d.buffer) || d.readErr != nil)
		if complete {
			d.offset += int64(p.at)
			d.buffer = d.buffer[p.at:]
			return decodeTarget(node, target, d.strict)
		}
		if p.err != nil {
			syntax, ok := p.err.(*SyntaxError)
			if !ok || syntax.message != "unexpected end of JSON input" {
				if ok {
					syntax.Offset += d.offset
				}
				d.err = p.err
				return d.err
			}
		}
		if d.readErr != nil {
			d.err = d.readErr
			if d.readErr == io.EOF && start < len(d.buffer) {
				d.err = io.ErrUnexpectedEOF
			}
			return d.err
		}
		var chunk [4096]byte
		n, err := d.reader.Read(chunk[:])
		if n < 0 || n > len(chunk) {
			d.err = errors.New("json: invalid reader count")
			return d.err
		}
		if n != 0 {
			d.buffer = append(d.buffer, chunk[:n]...)
			emptyReads = 0
		} else {
			emptyReads++
		}
		d.readErr = err
		if emptyReads >= 100 && err == nil {
			d.err = errors.New("json: reader made no progress")
			return d.err
		}
	}
}
