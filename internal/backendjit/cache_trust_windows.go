//go:build windows

package backendjit

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	windowsSEFileObject             = 1
	windowsOwnerSecurityInformation = 1
)

var windowsGetNamedSecurityInfo = syscall.NewLazyDLL("advapi32.dll").NewProc("GetNamedSecurityInfoW")

func cachedFileOwnedByCurrentUser(path string, info os.FileInfo) bool {
	if info == nil || path == "" {
		return false
	}
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	var owner *syscall.SID
	var descriptor uintptr
	status, _, _ := windowsGetNamedSecurityInfo.Call(
		uintptr(unsafe.Pointer(name)), windowsSEFileObject, windowsOwnerSecurityInformation,
		uintptr(unsafe.Pointer(&owner)), 0, 0, 0, uintptr(unsafe.Pointer(&descriptor)))
	if status != 0 || owner == nil || descriptor == 0 {
		return false
	}
	defer func() { _, _ = syscall.LocalFree(syscall.Handle(descriptor)) }()
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return false
	}
	defer func() { _ = token.Close() }()
	user, err := token.GetTokenUser()
	if err != nil || user == nil || user.User.Sid == nil {
		return false
	}
	ownerText, err := owner.String()
	if err != nil {
		return false
	}
	userText, err := user.User.Sid.String()
	return err == nil && ownerText == userText
}
