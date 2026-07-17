//go:build rtg && !darwin && !windows

package process

func Start(path, directory string) bool { return false }
