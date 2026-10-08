//go:build !renvo && !linux

package runtime

func (n *Native) EnablePerfMap() error { return nil }
func (n *Native) NameEntry(entry int, name string) error {
	err := n.arena.NameEntry(entry, name)
	if err != nil {
		n.SymbolErrors++
	}
	return err
}
