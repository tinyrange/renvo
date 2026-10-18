package engine

import emu "renvo.dev/internal/rfe/runtime"

// A fixed operation slab, divided into small allocation units. Admission is
// based on repeated compilation, not on guest execution: running a native
// helper many times does not itself need its source IR. New cold translations
// cannot flush helpers whose IR has actually been needed across arena resets.
// Every slice is revoked before its storage is reused. Native entries and
// PreparedBlock copies remain independent of this non-executable cache.
type translationCache struct {
	storage                 []emu.Op
	owners                  []*compiledBlock
	unit, next, used, count int
	scores                  [256]int // occupied allocation units per observed compilation count
	evictions               uint64
}

func (c *translationCache) recordCompile(block *compiledBlock) {
	old := block.irScore
	if old == 255 {
		return
	}
	block.irScore++
	if block.irSlot != 0 {
		c.scores[old] -= block.irUnits
		c.scores[block.irScore] += block.irUnits
	}
}
func (c *translationCache) forget(block *compiledBlock) {
	if block.irSlot == 0 {
		return
	}
	start, units := block.irSlot-1, block.irUnits
	if start < len(c.owners) && c.owners[start] == block {
		clear(c.owners[start : start+units])
		c.used -= units
		c.scores[block.irScore] -= units
		c.count--
	}
	block.irSlot, block.irUnits, block.ops = 0, 0, nil
}
func (c *translationCache) store(block *compiledBlock, ops []emu.Op) ([]emu.Op, bool) {
	if len(ops) == 0 || c.unit <= 0 || len(ops) > len(c.storage) {
		return ops, false
	}
	units := (len(ops) + c.unit - 1) / c.unit
	if units > len(c.owners) {
		return ops, false
	}
	c.forget(block)
	if block.irScore == 0 {
		block.irScore = 1
	}
	// Reject uncompetitive admissions before scanning the slab. A cache full
	// of equally or more frequently recompiled blocks is not a reason to
	// rescan or allocate more storage for every cold leaf.
	available := len(c.owners) - c.used
	for i := 0; i < int(block.irScore); i++ {
		available += c.scores[i]
	}
	if available < units {
		return ops, false
	}
	start, probes := 0, 0
	for {
		if c.next+units > len(c.owners) {
			c.next = 0
		}
		start = c.next
		blocked := false
		for i := start; i < start+units; i++ {
			owner := c.owners[i]
			if owner == nil {
				continue
			}
			probes++
			if probes > 64 {
				return ops, false
			} // optional cache work must stay cheaper than relowering
			if owner.irScore >= block.irScore {
				c.next = owner.irSlot - 1 + owner.irUnits
				blocked = true
				break
			}
			c.forget(owner)
			c.evictions++
		}
		if !blocked {
			break
		}
	}
	out := c.storage[start*c.unit : start*c.unit+len(ops)]
	copy(out, ops)
	for i := start; i < start+units; i++ {
		c.owners[i] = block
	}
	block.irSlot, block.irUnits = start+1, units
	c.scores[block.irScore] += units
	c.count++
	c.used += units
	c.next = start + units
	return out, true
}
