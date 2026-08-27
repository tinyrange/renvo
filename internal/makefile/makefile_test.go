package makefile

import "testing"

func TestParseAndPlanDependencyOrder(t *testing.T) {
	src := []byte("RENVO := renvo\nCFLAGS = -O2 -I include\nall: app\napp: main.o db.o\n\t$(RENVO) cc $^ -o $@\nmain.o: main.c\n\t@$(RENVO) cc $(CFLAGS) -c $< -o $@\ndb.o: db.c\n\t$(RENVO) cc $(CFLAGS) -c $< -o $@\n.PHONY: all\n")
	file, parseError := Parse(src)
	if parseError.Message != "" {
		t.Fatalf("Parse error = %#v", parseError)
	}
	existing := map[string]int64{"main.c": 1, "db.c": 1}
	commands, planError := Plan(file, nil, func(path string) (int64, bool) { stamp, ok := existing[path]; return stamp, ok })
	if planError.Message != "" {
		t.Fatalf("Plan error = %#v", planError)
	}
	if len(commands) != 3 {
		t.Fatalf("commands = %#v", commands)
	}
	want := []string{
		"renvo cc -O2 -I include -c main.c -o main.o",
		"renvo cc -O2 -I include -c db.c -o db.o",
		"renvo cc main.o db.o -o app",
	}
	for i := 0; i < len(want); i++ {
		if commands[i].Text != want[i] {
			t.Fatalf("command %d = %q, want %q", i, commands[i].Text, want[i])
		}
	}
	if !commands[0].Quiet {
		t.Fatal("@ recipe was not quiet")
	}
}

func TestRejectsShellRecipe(t *testing.T) {
	file, parseError := Parse([]byte("all:\n\tcc main.c -o app\n"))
	if parseError.Message != "" {
		t.Fatal(parseError)
	}
	_, planError := Plan(file, nil, nil)
	if planError.Message != "recipes must invoke renvo directly" {
		t.Fatalf("error = %#v", planError)
	}
}

func TestReportsDependencyCycle(t *testing.T) {
	file, _ := Parse([]byte("a: b\n\trenvo build b\nb: a\n\trenvo build a\n"))
	_, planError := Plan(file, []string{"a"}, nil)
	if planError.Message != "dependency cycle contains a" {
		t.Fatalf("error = %#v", planError)
	}
}
