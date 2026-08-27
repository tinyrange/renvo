//go:build renvo && !renvo_wasi_c_object

package driver

import "renvo.dev/internal/c11"

func cObjectHeaderPrelude(path string, source []byte, processed c11.PreprocessResult, reader cObjectIncludeReader, fs SourceFS) c11.HeaderResult {
	return c11.BuildObjectPrelude(path, source, reader)
}
