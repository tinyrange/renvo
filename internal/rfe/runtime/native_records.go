//go:build !renvo

package runtime

// nativeRecords is the sole host projection of validated operations into the
// backend-subset four-word record representation. Validation stays at callers.
func nativeRecords(ops []Op) []int {
	records := make([]int, 0, len(ops)*4)
	for _, op := range ops {
		records = append(records, op.Kind, int(op.A), int(op.B), int(op.Imm))
	}
	return records
}
