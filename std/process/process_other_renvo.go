//go:build renvo && !darwin && !windows

package process

func Start(path, directory string) bool { return false }
