package target

import "runtime"

var supported = []string{
	"linux/amd64",
	"linux/386",
	"linux/aarch64",
	"linux/arm",
	"windows/amd64",
	"windows/386",
	"wasi/wasm32",
}

func Default() string {
	host := runtime.GOOS + "/" + runtime.GOARCH
	if host == "linux/arm64" {
		return "linux/aarch64"
	}
	if Supported(host) {
		return host
	}
	return "linux/amd64"
}

func Supported(name string) bool {
	for i := 0; i < len(supported); i++ {
		supportedName := supported[i]
		if name == supportedName {
			return true
		}
	}
	return false
}

func List() string {
	out := ""
	for i := 0; i < len(supported); i++ {
		name := supported[i]
		if i > 0 {
			out = out + ", "
		}
		out = out + name
	}
	return out
}
