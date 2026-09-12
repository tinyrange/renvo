package rfe

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestTypedLowering(t *testing.T) {
	header := "package demo\n//rfe:word 8\n//rfe:state 4\n//rfe:instruction 254 0\n"
	for _, body := range []string{
		"guard(1)", "state[0]=true", "state[state[0]]=0", "state[0]=1<<state[1]",
		"state[0]=true+1", "guard(opcode==true)", "state[0]=choose(true,0,false)",
	} {
		if _, err := GenerateLowering([]byte(header + "func example(opcode uint64){" + body + "}")); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	source := string(testSource("demo")) + "-- instructions.lower --\n" + header + `func example(opcode uint64) {
 guard(opcode==0 || opcode==1)
 index:=opcode&1
 a:=state[index]
 state[2]=choose(a>=1 && a<=3,u8(a+256),0-1)
 state[3]=a<<(opcode+64)
}` + "\n-- semantics_test.go --\n" + `package demo
import("testing"; emu "renvo.dev/internal/rfe/runtime")
func TestSemantics(t *testing.T){
 for opcode:=uint64(0);opcode<2;opcode++ {
  for a:=uint64(0);a<5;a++ {
   state:=[4]uint64{};state[opcode]=a
   interpreted:=state
   if !Execute(opcode,&state){t.Fatal("decoder")}
   want:=uint64(0xffffffffffffffff);if a>=1&&a<=3{want=a}
   if state[2]!=want||state[3]!=0{t.Fatalf("Go semantics: %v",state)}
   var b emu.Builder
   if !Lower(opcode,&b){t.Fatal("lowering")}
   if err:=emu.Interpret(b.Finish(4),interpreted[:]);err!=nil{t.Fatal(err)}
   if state!=interpreted{t.Fatalf("Go/IR disagreement: %v %v",state,interpreted)}
  }
 }
}
`
	packages, err := Resolve([]byte(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	root, _ := filepath.Abs("../..")
	dir := t.TempDir()
	if err = Workspace(dir, root, packages); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", "./packages/...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOSUMDB=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated semantics: %v\n%s", err, out)
	}
}

func TestResolveLoadsSharedDependencyOnce(t *testing.T) {
	files := map[string][]byte{"a": testSource("a", Dependency{Name: "cpu"}), "b": testSource("b", Dependency{Name: "cpu"}), "cpu": testSource("cpu")}
	loads := map[string]int{}
	root := testSource("root", Dependency{Name: "a"}, Dependency{Name: "b"})
	_, err := Resolve(root, func(name string) ([]byte, error) { loads[name]++; return files[name], nil })
	if err != nil || loads["cpu"] != 1 {
		t.Fatalf("loads %v: %v", loads, err)
	}
	p, _ := Decode(files["cpu"])
	files["b"] = testSource("b", Dependency{Name: "cpu", SHA256: strings.Repeat("0", len(p.Digest))})
	if _, err = Resolve(root, mapLoader(files)); err == nil {
		t.Fatal("cached dependency bypassed hash check")
	}
}
