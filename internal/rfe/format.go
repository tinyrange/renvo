package rfe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"sort"
	"strings"
)

// Format canonicalizes an editable .rfe archive, including its embedded Go
// and instruction DSL. Section names and non-code contents are preserved.
func Format(source []byte) ([]byte, error) {
	if !bytes.HasPrefix(source, []byte("RFE 1\n")) {
		return nil, fmt.Errorf("formatter requires an RFE 1 text archive")
	}
	p, err := Decode(source)
	if err != nil {
		return nil, err
	}
	sort.Slice(p.Manifest.Requires, func(i, j int) bool { return p.Manifest.Requires[i].Name < p.Manifest.Requires[j].Name })
	manifest, err := json.MarshalIndent(p.Manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	p.Files["rfe.json"] = manifest
	names := make([]string, 0, len(p.Files))
	for name := range p.Files {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i] == "rfe.json" {
			return true
		}
		if names[j] == "rfe.json" {
			return false
		}
		return names[i] < names[j]
	})
	var out bytes.Buffer
	out.WriteString("RFE 1\n")
	for _, name := range names {
		data := p.Files[name]
		if strings.HasSuffix(name, ".go") || strings.HasSuffix(name, ".lower") {
			data, err = format.Source(data)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			if strings.HasSuffix(name, ".lower") {
				if _, err = GenerateLowering(data); err != nil {
					return nil, fmt.Errorf("%s: %w", name, err)
				}
			}
		}
		fmt.Fprintf(&out, "\n-- %s --\n", name)
		out.Write(bytes.TrimRight(data, "\r\n"))
		out.WriteByte('\n')
	}
	return out.Bytes(), nil
}
