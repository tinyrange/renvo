package runtime

// RegionGuard is an effect-only side exit after an architectural checkpoint.
// A failed condition reports RegionExit, not a fault or an instruction to replay.
const RegionGuard = 26
const RegionExit = 4

func (b *Builder) RegionGuard(condition Value) {
	if b.Ops[condition].Kind == Const && b.Ops[condition].Imm != 0 {
		return
	}
	b.emit(Op{Kind: RegionGuard, A: condition})
}
