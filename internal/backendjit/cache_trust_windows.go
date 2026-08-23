//go:build windows

package backendjit

import "os"

func cachedFileOwnedByCurrentUser(info os.FileInfo) bool {
	return info != nil
}
