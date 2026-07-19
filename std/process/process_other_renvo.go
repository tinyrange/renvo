//go:build renvo && !darwin && !windows && !browser

package process

func Start(path, directory string) bool { return false }
