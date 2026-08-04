//go:build !renvo && ((!linux && !windows && !darwin) || (windows && !amd64 && !386 && !arm64) || (darwin && !arm64))

package runimage

import (
	"os"
	"runtime"
)

func prepareNativeExecutable(data []byte) (preparedExecutable, error) {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	file, err := os.CreateTemp("", "renvo-script-*"+suffix)
	if err != nil {
		return preparedExecutable{}, err
	}
	path := file.Name()
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(path)
	}
	if _, err = file.Write(data); err != nil {
		cleanup()
		return preparedExecutable{}, err
	}
	if err = file.Chmod(0o700); err != nil {
		cleanup()
		return preparedExecutable{}, err
	}
	if err = file.Close(); err != nil {
		_ = os.Remove(path)
		return preparedExecutable{}, err
	}
	return preparedExecutable{
		path:   path,
		close:  func() { _ = os.Remove(path) },
		loader: "native-file",
	}, nil
}
