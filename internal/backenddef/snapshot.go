package backenddef

import (
	"path"
	"renvo.dev/internal/rtg"
)

type definitionSnapshotLoader struct{ files []rtg.ImportSource }

func (loader definitionSnapshotLoader) LoadImport(filename, imported string) rtg.ImportSource {
	wanted := path.Clean(path.Join(path.Dir(filename), imported))
	for _, file := range loader.files {
		if file.Filename == wanted {
			return file
		}
	}
	return rtg.ImportSource{}
}
