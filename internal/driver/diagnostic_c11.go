package driver

import "renvo.dev/internal/c11"

func c11ErrorDiagnostic(diagnostic Diagnostic, detail int) Diagnostic {
	diagnostic.Phase = "c11"
	switch detail {
	case c11.TranslateErrScan:
		diagnostic.Code, diagnostic.Message = "RENVO-C11-001", "invalid C token or unterminated comment/literal"
	case c11.TranslateErrVLA:
		diagnostic.Code, diagnostic.Message = "RENVO-C11-002", "variable length arrays are not supported"
	case c11.TranslateErrDeclaration:
		diagnostic.Code, diagnostic.Message = "RENVO-C11-003", "invalid C declaration"
	case c11.TranslateErrStatement:
		diagnostic.Code, diagnostic.Message = "RENVO-C11-004", "invalid C statement or expression"
	case c11.TranslateErrUnsupported:
		diagnostic.Code, diagnostic.Message = "RENVO-C11-005", "unsupported C language construct"
	default:
		diagnostic.Phase, diagnostic.Code = "compiler", "RENVO-BUG-007"
		diagnostic.Message = "compiler bug: C translator returned undeclared error code " + diagnosticIntText(detail)
	}
	return diagnostic
}
