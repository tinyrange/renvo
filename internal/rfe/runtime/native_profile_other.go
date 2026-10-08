//go:build !renvo && !linux

package runtime

func (n *Native) EnablePerfMap() error                   { return nil }
func (n *Native) NameEntry(entry int, name string) error { return nil }
