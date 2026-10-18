package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestWasmHostInvokeParsedSyscallArgumentsAndNullSlot(t *testing.T) {
	source, err := os.ReadFile("tests/wasm_host_invoke.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	image, ok := RenvoCompileSourceToBytes(source, "wasi/wasm32")
	if !ok {
		t.Fatal("host syscall source did not compile")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node required for actual compiled-Wasm regression")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node, "-e", `
const fs=require('node:fs'),code=fs.readFileSync(0),module=new WebAssembly.Module(code),imports=WebAssembly.Module.imports(module);
if(!imports.some(i=>i.module==='renvo_rfe'&&i.name==='gates'&&i.kind==='table'))throw Error('missing table import');
const gates=new WebAssembly.Table({element:'anyfunc',initial:32768,maximum:32768});
// A five-word typed Wasm callee returns the sum, proving argument placement.
const leaf=new Uint8Array([0,97,115,109,1,0,0,0,1,10,1,96,5,127,127,127,127,127,1,127,3,2,1,0,7,10,1,6,105,110,118,111,107,101,0,0,10,18,1,16,0,32,0,32,1,106,32,2,106,32,3,106,32,4,106,11]);
gates.set(1,new WebAssembly.Instance(new WebAssembly.Module(leaf)).exports.invoke);
let instance,output='',exit=null;const view=()=>new DataView(instance.exports.memory.buffer);
const wasi={};for(const i of imports)if(i.module==='wasi_snapshot_preview1')wasi[i.name]=()=>0;
for(const name of ['args_sizes_get','environ_sizes_get'])wasi[name]=(a,b)=>{view().setUint32(a,0,true);view().setUint32(b,0,true);return 0};
wasi.fd_write=(fd,at,n,out)=>{let size=0;for(let i=0;i<n;i++){const p=view().getUint32(at+8*i,true),len=view().getUint32(at+8*i+4,true);output+=Buffer.from(instance.exports.memory.buffer,p,len).toString();size+=len}view().setUint32(out,size,true);return 0};
wasi.proc_exit=code=>{exit=code;throw 'exit'};
instance=new WebAssembly.Instance(module,{wasi_snapshot_preview1:wasi,renvo_rfe:{gates}});
try{instance.exports._start()}catch(e){if(e!=='exit')throw e}
if(exit!==0||output!=='PASS\n')throw Error(JSON.stringify({exit,output}));console.log('PASS');`)
	cmd.Stdin = bytes.NewReader(image)
	out, err := cmd.CombinedOutput()
	if err != nil || string(out) != "PASS\n" {
		t.Fatalf("compiled Wasm host invocation: %v\n%s", err, out)
	}
}

func TestWasmDynamicAndOrdinarySyscallsNeedNoGateImport(t *testing.T) {
	for _, source := range []string{
		"package main\n//renvo:intrinsic renvo_runtime_Syscall\nfunc hostCall(number,fd,at,n int) int{return -1}\nfunc appMain() int { return hostCall(217, 1, 0, 0) }",
		"package main\n//renvo:intrinsic renvo_runtime_Syscall\nfunc hostCall(number,a,b,c,d,e int) int{return -1}\nfunc invoke(number int) int { return hostCall(number, 1, 16, 64, 3, 128) }; func appMain() int { return invoke(217) }",
	} {
		image, ok := RenvoCompileSourceToBytes([]byte(source), "wasi/wasm32")
		if !ok {
			t.Fatal("ordinary syscall did not compile")
		}
		if bytes.Contains(image, []byte("renvo_rfe")) {
			t.Fatal("ordinary/dynamic syscall acquired optional host dependency")
		}
	}
}
