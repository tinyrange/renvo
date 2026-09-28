package json

import (
	"io"
	"testing"
)

type chunkReader struct {
	data        string
	width       int
	eofWithData bool
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := r.width
	if n > len(r.data) {
		n = len(r.data)
	}
	if n > len(p) {
		n = len(p)
	}
	copy(p, r.data[:n])
	r.data = r.data[n:]
	if r.eofWithData && len(r.data) == 0 {
		return n, io.EOF
	}
	return n, nil
}

func TestDecoderStreaming(t *testing.T) {
	for width := 1; width <= 32; width++ {
		d := NewDecoder(&chunkReader{data: " 12.5e1 \"escaped\\ntext\" [null,true] {}  ", width: width, eofWithData: width%2 == 0})
		var value any
		if err := d.Decode(&value); err != nil || value != float64(125) {
			t.Fatal("number", err)
		}
		if d.InputOffset() != 7 {
			t.Fatal("input offset")
		}
		if err := d.Decode(&value); err != nil || value != "escaped\ntext" {
			t.Fatal("string", err)
		}
		if err := d.Decode(&value); err != nil {
			t.Fatal(err)
		}
		items, ok := value.([]any)
		if !ok || len(items) != 2 || items[0] != nil || items[1] != true {
			t.Fatal("array")
		}
		if err := d.Decode(&value); err != nil {
			t.Fatal(err)
		}
		if err := d.Decode(&value); err != io.EOF {
			t.Fatal("EOF", err)
		}
		if err := d.Decode(&value); err != io.EOF {
			t.Fatal("repeated EOF", err)
		}
	}
}

func TestDecoderRejectsTrailingAndIncomplete(t *testing.T) {
	for _, source := range []string{"[", "{\"a\":", "\"missing", "1e", "tru"} {
		d := NewDecoder(&chunkReader{data: source, width: 1})
		var value any
		if err := d.Decode(&value); err != io.ErrUnexpectedEOF {
			t.Fatal("incomplete", source, err)
		}
	}
	d := NewDecoder(&chunkReader{data: "{} invalid", width: 32})
	var value any
	if err := d.Decode(&value); err != nil {
		t.Fatal(err)
	}
	if err := d.Decode(&value); err == nil || err == io.EOF {
		t.Fatal("trailing data accepted")
	}
	d = NewDecoder(&chunkReader{data: "{\"unknown\":1} {\"name\":\"next\"}", width: 3})
	d.DisallowUnknownFields()
	var record decodeChild
	if err := d.Decode(&record); err == nil {
		t.Fatal("unknown field accepted")
	}
	if err := d.Decode(&record); err != nil || record.Name != "next" {
		t.Fatal("conversion error did not consume value", err)
	}
}
