package engine

// The same 4096 observation slots are four-way associative. Full-PC tags
// prevent mutually colliding hot instruction boundaries from resetting each
// other's heat indefinitely. This table grants discovery, never execution.
func (e *Engine) discoveryObservation(pc uint64, create bool) *hotObservation {
	set := ((pc / e.arch.Alignment) ^ (pc >> 12)) & 1023
	var empty *hotObservation
	for way := uint64(0); way < 4; way++ {
		o := &e.hot[set+way*1024]
		if o.visits != 0 && o.pc == pc {
			return o
		}
		if o.visits == 0 && empty == nil {
			empty = o
		}
	}
	if !create {
		return nil
	}
	if empty == nil {
		empty = &e.hot[set+(e.hotVictim&3)*1024]
		e.hotVictim++
	}
	*empty = hotObservation{pc: pc}
	return empty
}
