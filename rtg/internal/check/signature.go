package check

import "j5.nz/rtg/rtg/internal/syntax"

type FuncSignature struct {
	Receiver []Field
	Params   []Field
	Results  []Field
}

type Field struct {
	Name      string
	NameTok   int
	TypeStart int
	TypeEnd   int
	Variadic  bool
}

func LookupField(fields []Field, name string) int {
	for i := 0; i < len(fields); i++ {
		if fields[i].Name == name {
			return i
		}
	}
	return -1
}

func buildFuncSignature(file syntax.File, fn syntax.FuncDecl) FuncSignature {
	return buildSignatureFromParts(file, fn.ReceiverStart, fn.ReceiverEnd, fn.ParamsStart, fn.ParamsEnd, fn.ResultStart, fn.ResultEnd)
}

func buildSignatureFromParts(file syntax.File, receiverStart int, receiverEnd int, paramsStart int, paramsEnd int, resultStart int, resultEnd int) FuncSignature {
	var sig FuncSignature
	if receiverStart >= 0 && receiverEnd > receiverStart {
		sig.Receiver = parseFieldList(file, receiverStart, receiverEnd)
	}
	if paramsStart >= 0 && paramsEnd > paramsStart {
		sig.Params = parseFieldList(file, paramsStart+1, paramsEnd-1)
	}
	if resultStart >= 0 && resultEnd > resultStart {
		if tokCharIs(file, resultStart, '(') {
			end := resultEnd - 1
			if tokCharIs(file, end, ')') {
				sig.Results = parseFieldList(file, resultStart+1, end)
			}
		} else {
			start, end := trimFieldSpan(file, resultStart, resultEnd)
			if start < end {
				sig.Results = append(sig.Results, Field{NameTok: -1, TypeStart: start, TypeEnd: end, Variadic: fieldIsVariadic(file, start)})
			}
		}
	}
	return sig
}

func parseFieldList(file syntax.File, start int, end int) []Field {
	capacity := end - start
	if capacity < 0 {
		capacity = 0
	}
	fields := make([]Field, 0, capacity)
	pending := make([]int, 0, capacity)
	i := start
	for i < end {
		segEnd := nextTopLevelComma(file, i, end)
		first, last := trimFieldSpan(file, i, segEnd)
		if first >= last {
			i = segEnd + 1
			continue
		}
		if isSingleIdent(file, first, last) {
			pending = append(pending, first)
			i = segEnd + 1
			continue
		}
		if file.Tokens[first].Kind == syntax.TokenIdent && first+1 < last && !tokCharIs(file, first+1, '.') {
			fields = appendNamedFields(fields, file, pending, first, first+1, last)
			pending = pending[:0]
		} else {
			fields = appendPendingUnnamed(fields, file, pending)
			pending = pending[:0]
			fields = append(fields, Field{NameTok: -1, TypeStart: first, TypeEnd: last, Variadic: fieldIsVariadic(file, first)})
		}
		i = segEnd + 1
	}
	return appendPendingUnnamed(fields, file, pending)
}

func appendNamedFields(fields []Field, file syntax.File, pending []int, current int, typeStart int, typeEnd int) []Field {
	for i := 0; i < len(pending); i++ {
		fields = append(fields, Field{
			Name:      tokenString(file, pending[i]),
			NameTok:   pending[i],
			TypeStart: typeStart,
			TypeEnd:   typeEnd,
			Variadic:  fieldIsVariadic(file, typeStart),
		})
	}
	fields = append(fields, Field{
		Name:      tokenString(file, current),
		NameTok:   current,
		TypeStart: typeStart,
		TypeEnd:   typeEnd,
		Variadic:  fieldIsVariadic(file, typeStart),
	})
	return fields
}

func appendPendingUnnamed(fields []Field, file syntax.File, pending []int) []Field {
	for i := 0; i < len(pending); i++ {
		fields = append(fields, Field{NameTok: -1, TypeStart: pending[i], TypeEnd: pending[i] + 1, Variadic: fieldIsVariadic(file, pending[i])})
	}
	return fields
}

func trimFieldSpan(file syntax.File, start int, end int) (int, int) {
	for start < end && isFieldSeparator(file, start) {
		start++
	}
	for end > start && isFieldSeparator(file, end-1) {
		end--
	}
	return start, end
}

func isFieldSeparator(file syntax.File, tok int) bool {
	return tokCharIs(file, tok, ',') || tokCharIs(file, tok, ';')
}

func isSingleIdent(file syntax.File, start int, end int) bool {
	return end-start == 1 && file.Tokens[start].Kind == syntax.TokenIdent
}

func fieldIsVariadic(file syntax.File, start int) bool {
	return tokenTextIs(file, start, "...")
}
