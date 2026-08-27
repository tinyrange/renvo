//go:build !renvo || renvo_wasi_c_object

package driver

import "renvo.dev/internal/c11"

func cObjectHeaderPrelude(path string, source []byte, processed c11.PreprocessResult, reader cObjectIncludeReader, fs SourceFS) c11.HeaderResult {
	headers := make([]c11.HeaderSource, 0, len(processed.Dependencies))
	for i := 0; i < len(processed.Dependencies); i++ {
		src, ok := fs.ReadFile(processed.Dependencies[i])
		if !ok {
			return c11.HeaderResult{Ok: false, ErrorPath: processed.Dependencies[i], ErrorAt: -1}
		}
		headers = append(headers, c11.HeaderSource{Path: processed.Dependencies[i], Source: src})
	}
	return c11.BuildObjectPrelude(path, processed.Source, reader, headers...)
}
