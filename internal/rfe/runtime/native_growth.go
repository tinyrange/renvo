//go:build !renvo

package runtime

func (n *Native) AllowGrowth()      { n.arena.AllowGrowth() }
func (n *Native) Release(entry int) { n.Bytes -= n.arena.Release(entry) }
