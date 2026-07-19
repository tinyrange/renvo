package main

type miniStmt struct {
	kind      int
	startTok  int
	endTok    int
	exprStart int
	exprEnd   int
	bodyStart int
	bodyEnd   int
	elseStart int
	elseEnd   int
	nameStart int
	nameEnd   int
}

type miniParse struct {
	stmts []miniStmt
}

func addMiniStmt(bp *miniParse, kind int, startTok int, endTok int, exprStart int, exprEnd int, bodyStart int, bodyEnd int, elseStart int, elseEnd int, nameStart int, nameEnd int) {
	var stmt miniStmt
	stmt.kind = kind
	stmt.startTok = startTok
	stmt.endTok = endTok
	stmt.exprStart = exprStart
	stmt.exprEnd = exprEnd
	stmt.bodyStart = bodyStart
	stmt.bodyEnd = bodyEnd
	stmt.elseStart = elseStart
	stmt.elseEnd = elseEnd
	stmt.nameStart = nameStart
	stmt.nameEnd = nameEnd
	bp.stmts = append(bp.stmts, stmt)
}

func appMain() int {
	var bp miniParse
	bp.stmts = make([]miniStmt, 0, 4)
	addMiniStmt(&bp, 1, 8, 14, 9, 14, 20, 21, 30, 31, 40, 41)
	if len(bp.stmts) == 1 {
		stmt := bp.stmts[0]
		if stmt.kind == 1 && stmt.startTok == 8 && stmt.endTok == 14 && stmt.exprStart == 9 && stmt.exprEnd == 14 && stmt.bodyStart == 20 && stmt.bodyEnd == 21 && stmt.elseStart == 30 && stmt.elseEnd == 31 && stmt.nameStart == 40 && stmt.nameEnd == 41 {
			print("PASS\n")
		}
	}
	return 0
}
