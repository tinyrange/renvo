//go:build renvo && browser && wasm32

package process

func Start(path, directory string) bool {
	message := []byte(path)
	return len(message) > 0 && write(4, message, -1) == len(message)
}
