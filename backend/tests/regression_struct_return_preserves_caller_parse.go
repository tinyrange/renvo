package main

type miniExpr struct {
	kind      int
	nameStart int
	nameEnd   int
}

type miniParse struct {
	exprs []miniExpr
	args  []int
	ok    bool
}

type miniResult struct {
	value int
	ok    bool
}

var fixedState int
var fixedValue int

func addExpr(ep *miniParse, kind int, start int, end int) int {
	idx := len(ep.exprs)
	ep.exprs = append(ep.exprs, miniExpr{kind: kind, nameStart: start, nameEnd: end})
	return idx
}

func parseInto(ep *miniParse, kind int, start int, end int) {
	ep.exprs = make([]miniExpr, 0, 4)
	ep.args = make([]int, 0, 4)
	ep.ok = true
	addExpr(ep, kind, start, end)
}

func evalConst() miniResult {
	var ep miniParse
	parseInto(&ep, 77, 90, 110)
	if !ep.ok || len(ep.exprs) != 1 {
		return miniResult{}
	}
	return miniResult{value: ep.exprs[0].kind, ok: true}
}

func loadFixed() {
	if fixedState != 0 {
		return
	}
	fixedState = -1
	r := evalConst()
	if r.ok {
		fixedState = 1
		fixedValue = r.value
	}
}

func useParsedAfterLoad() int {
	var ep miniParse
	parseInto(&ep, 11, 20, 30)
	if !ep.ok || len(ep.exprs) != 1 {
		return 1
	}
	loadFixed()
	if len(ep.exprs) != 1 {
		return 2
	}
	if ep.exprs[0].kind != 11 {
		return 3
	}
	if ep.exprs[0].nameStart != 20 || ep.exprs[0].nameEnd != 30 {
		return 4
	}
	return 0
}

func appMain(args []string) int {
	fixedState = 0
	fixedValue = 0
	status := useParsedAfterLoad()
	if status == 1 {
		print("RENVO-1120 caller parse precheck failed\n")
		return 1
	}
	if status == 2 {
		print("RENVO-1120 caller parse len clobbered\n")
		return 1
	}
	if status == 3 {
		print("RENVO-1120 caller parse kind clobbered\n")
		return 1
	}
	if status == 4 {
		print("RENVO-1120 caller parse range clobbered\n")
		return 1
	}
	if status != 0 {
		print("RENVO-1120 caller parse unknown clobber\n")
		return 1
	}
	if fixedValue != 77 {
		print("RENVO-1120 fixed value not loaded\n")
		return 1
	}
	print("PASS\n")
	return 0
}
