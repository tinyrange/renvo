package runtime

import "fmt"

// Only graphs share an epilogue. Ordinary leaves and resident loops preserve
// their original exits. These three private locals are never guest pointers.
type wasmGraphExit struct {
	status, address, hasAddress int
}

// Rewrite generated returns into branches to a newly surrounding exit block.
// Parsing operands is essential: a byte equal to return may occur in a local,
// constant, memory offset or branch table. Existing relative branches retain
// their destinations because the new block surrounds their original targets.
// This accepts only the integer emitter's instruction subset, failing closed
// if a future emitter introduces an instruction with an unhandled immediate.
func wasmGraphExitBranches(body []byte) ([]byte, error) {
	var out wasmCode
	depth := 0
	for pos := 0; pos < len(body); {
		start := pos
		op := body[pos]
		pos++
		immediates := 0
		switch {
		case op == 0x02 || op == 0x03 || op == 0x04:
			if pos >= len(body) || body[pos] != 0x40 && body[pos] != 0x7e && body[pos] != 0x7f {
				return nil, fmt.Errorf("invalid WASM graph block type")
			}
			pos++
			depth++
		case op == 0x05:
			if depth == 0 {
				return nil, fmt.Errorf("invalid WASM graph else")
			}
		case op == 0x0b:
			if depth == 0 {
				return nil, fmt.Errorf("unbalanced WASM graph end")
			}
			depth--
		case op == 0x0f:
			out.byte(0x0c)
			out.u32(uint32(depth))
			continue
		case op == 0x0c || op == 0x0d || op == 0x10 || op >= 0x20 && op <= 0x23:
			immediates = 1
		case op == 0x11 || op >= 0x28 && op <= 0x3e:
			immediates = 2
		case op == 0x41 || op == 0x42:
			// Signed LEB has the same continuation bit as unsigned LEB.
			immediates = 1
		case op == 0x0e:
			count, end, ok := wasmGraphLEB(body, pos, 5)
			if !ok || uint64(count)+1 > uint64(len(body)-end) {
				return nil, fmt.Errorf("invalid WASM graph branch table")
			}
			pos = end
			immediates = int(count) + 1
		case op == 0x00 || op == 0x01 || op == 0x1a || op == 0x1b || op >= 0x45 && op <= 0xc4:
		default:
			return nil, fmt.Errorf("unsupported WASM graph opcode %d", op)
		}
		for i := 0; i < immediates; i++ {
			limit := 5
			if op == 0x42 {
				limit = 10
			}
			_, end, ok := wasmGraphLEB(body, pos, limit)
			if !ok {
				return nil, fmt.Errorf("invalid WASM graph immediate")
			}
			pos = end
		}
		out.code = append(out.code, body[start:pos]...)
	}
	if depth != 0 {
		return nil, fmt.Errorf("unbalanced WASM graph body")
	}
	return out.code, nil
}

func wasmGraphLEB(body []byte, pos, limit int) (uint32, int, bool) {
	value := uint32(0)
	for n := 0; n < limit && pos < len(body); n++ {
		b := body[pos]
		pos++
		if n < 5 {
			value |= uint32(b&127) << uint(n*7)
		}
		if b&128 == 0 {
			return value, pos, true
		}
	}
	return 0, pos, false
}
