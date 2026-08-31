//go:build renvo && (wasi || wasip1 || browser || vm || esp32p4)

package driver

func renvoRunTarget() string { return "" }
func renvoRunTargetID() int  { return 0 }

func RunNativeLinkedImage(native []byte, script string, args []string, env []string) int {
	return -1
}
