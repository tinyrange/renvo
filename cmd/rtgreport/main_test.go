package main

import "testing"

func TestOutcomeMatches(t *testing.T) {
	tests := []struct {
		expected string
		observed string
		want     bool
	}{
		{"accepted", "accepted", true},
		{"frontend-diagnostic", "frontend-diagnostic", true},
		{"excluded", "frontend-diagnostic", true},
		{"excluded", "accepted", false},
		{"backend-failure", "frontend-diagnostic", false},
	}
	for _, test := range tests {
		if got := outcomeMatches(test.expected, test.observed); got != test.want {
			t.Errorf("outcomeMatches(%q, %q) = %v, want %v", test.expected, test.observed, got, test.want)
		}
	}
}

func TestFrontendDiagnosticClassification(t *testing.T) {
	if !isFrontendDiagnostic("rtg: frontend pipeline failed at package=0 file=0 token=3") {
		t.Fatal("frontend pipeline error was not classified as a frontend diagnostic")
	}
	if !isFrontendDiagnostic("./cmd/app/main.go:5:2: error RTG-CHECK-002 (checker): duplicate declaration") {
		t.Fatal("structured checker error was not classified as a frontend diagnostic")
	}
	if isFrontendDiagnostic("rtg: error RTG-BACKEND-003 (backend): compiler failed") {
		t.Fatal("structured backend error was classified as a frontend diagnostic")
	}
	if isFrontendDiagnostic("rtg: backend compilation failed") {
		t.Fatal("backend failure was classified as a frontend diagnostic")
	}
}
