//go:build !renvo

package backendjit

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"

	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgb"
)

func nativeCompilerCachePath(directory string, key string) string {
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	return filepath.Join(directory, key+suffix)
}

func loadNativeCompilerCache(directory string, key string, descriptor rtg.TargetDescriptor, host string) (Prepared, bool) {
	cache := FileCache{Directory: directory}
	manifestPath := filepath.Join(directory, key+".rtgb")
	if !trustedCachedFile(manifestPath, false) {
		return Prepared{}, false
	}
	encoded, found := cache.Load(key)
	if !found {
		return Prepared{}, false
	}
	manifest, ok := rtgb.Decode(encoded)
	if !ok || !seededCompatible(manifest, descriptor, host) || len(manifest.Payload) != sha256.Size {
		return Prepared{}, false
	}
	path := nativeCompilerCachePath(directory, key)
	if !trustedCachedExecutable(path) {
		return Prepared{}, false
	}
	executable, err := os.ReadFile(path)
	if err != nil || !validHostExecutable(executable, host) {
		return Prepared{}, false
	}
	digest := sha256.Sum256(executable)
	if !bytes.Equal(manifest.Payload, digest[:]) {
		return Prepared{}, false
	}
	manifest.Payload = executable
	full, ok := rtgb.Encode(manifest)
	if !ok {
		return Prepared{}, false
	}
	return Prepared{
		Artifact: manifest, Encoded: full, CachePath: filepath.Join(directory, key+".rtgb"),
		ExecutablePath: path, CacheHit: true, Ok: true,
	}, true
}

func storeNativeCompilerCache(directory string, key string, artifact rtgb.Artifact) (Prepared, error) {
	path := nativeCompilerCachePath(directory, key)
	if !validHostExecutable(artifact.Payload, artifact.Host) {
		return Prepared{}, os.ErrInvalid
	}
	digest := sha256.Sum256(artifact.Payload)
	manifest := artifact
	manifest.Payload = digest[:]
	encoded, ok := rtgb.Encode(manifest)
	if !ok {
		return Prepared{}, os.ErrInvalid
	}
	// The executable is published first. The manifest is the commit record, so
	// readers never execute a file without a matching compatibility identity and
	// content digest.
	if err := publishMode(path, artifact.Payload, 0o700); err != nil {
		return Prepared{}, err
	}
	if err := (FileCache{Directory: directory}).Store(key, encoded); err != nil {
		return Prepared{}, err
	}
	if !trustedCachedExecutable(path) || !trustedCachedFile(filepath.Join(directory, key+".rtgb"), false) {
		return Prepared{}, os.ErrPermission
	}
	full, ok := rtgb.Encode(artifact)
	if !ok {
		return Prepared{}, os.ErrInvalid
	}
	return Prepared{
		Artifact: artifact, Encoded: full, CachePath: filepath.Join(directory, key+".rtgb"),
		ExecutablePath: path, Ok: true,
	}, nil
}

func trustedCachedExecutable(path string) bool {
	return trustedCachedFile(path, true)
}

func trustedCachedFile(path string, executable bool) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	if !cachedFileOwnedByCurrentUser(info) || info.Mode().Perm()&0o022 != 0 ||
		executable && info.Mode().Perm()&0o111 == 0 {
		return false
	}
	directory, err := os.Stat(filepath.Dir(path))
	return err == nil && directory.IsDir() && cachedFileOwnedByCurrentUser(directory) && directory.Mode().Perm()&0o022 == 0
}

func validHostExecutable(source []byte, host string) bool {
	if len(source) < 4 {
		return false
	}
	switch host {
	case "linux/amd64", "linux/386", "linux/aarch64", "linux/arm",
		"freebsd/amd64", "openbsd/amd64", "netbsd/amd64":
		return bytes.Equal(source[:4], []byte{0x7f, 'E', 'L', 'F'})
	case "windows/amd64", "windows/386", "windows/arm64":
		return source[0] == 'M' && source[1] == 'Z'
	case "darwin/arm64":
		return bytes.Equal(source[:4], []byte{0xcf, 0xfa, 0xed, 0xfe}) ||
			bytes.Equal(source[:4], []byte{0xfe, 0xed, 0xfa, 0xcf})
	}
	return false
}
