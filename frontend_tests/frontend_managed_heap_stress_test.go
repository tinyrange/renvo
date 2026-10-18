package frontend_tests

var managedHeapStressCases = []struct{ name, limit, source string }{
	{"page-index-interior-roots", "16MiB", `package main
import "renvo.dev/x/runtime"
var roots [80]*byte
func build(){
 for i:=0;i<len(roots);i++{
  size:=4080+(i%7)*4096
  if i==0{size=3*1024*1024}
  b:=make([]byte,size)
  b[size-1]=byte(i+17)
  roots[i]=&b[size-1]
 }
}
func churn(){for i:=0;i<160;i++{b:=make([]byte,7000+(i%9)*997);b[len(b)-1]=91}}
func check(){for i,p:=range roots{if p!=nil&&*p!=byte(i+17){panic("interior root lost")}}}
func main(){
 build();churn();runtime.Collect();check()
 for round:=0;round<8;round++{
  for i:=1;i<len(roots);i+=2{roots[i]=nil}
  runtime.Collect();churn();check()
  for i:=1;i<len(roots);i+=2{b:=make([]byte,9000+(i%7)*4096);for _,v:=range b{if v!=0{panic("reuse not zero")}};b[len(b)-1]=byte(i+17);roots[i]=&b[len(b)-1]}
  runtime.Collect();check()
 }
 println("PASS")
}
`},
	{"oversized-requests", "64KiB", `package main
import "renvo.dev/x/runtime"
var sizes=[5]int{-1,9223372036854775807,4611686018427387904,65536,1000000}
var data []byte
var count int
var allocated bool
func attempt(size int){defer func(){if recover()==nil{panic("missing failure")};count++}();data=make([]byte,size);allocated=true}
func main(){live:=make([]byte,100);live[99]=72;for _,size:=range sizes{attempt(size);runtime.Collect();if live[99]!=72{panic("lost root")}};if count!=5||allocated{panic("bad oversized allocation")};println("PASS")}
`},
	{"zero-length-capacity-root", "64KiB", `package main
import "renvo.dev/x/runtime"
func makeTail()[]byte{b:=make([]byte,4000);b[3999]=78;return b[3001:3001]}
func waste(){b:=make([]byte,5000);b[4999]=33}
func main(){tail:=makeTail();for i:=0;i<100;i++{waste()};runtime.Collect();if len(tail)!=0||cap(tail)!=999{panic("slice header")};tail=tail[:999];if tail[998]!=78{panic("zero length root")};println("PASS")}
`},
	{"random-cyclic-graphs", "128KiB", `package main
import "renvo.dev/x/runtime"
type Node struct{left,right *Node; id int; data []byte}
var root *Node
var expected int
var seed uint32=1
func next()uint32{seed=seed*1664525+1013904223;return seed}
func checksum()int{var seen [96]bool;var queue [256]*Node;queue[0]=root;tail:=1;sum:=0;for head:=0;head<tail;head++{n:=queue[head];if n==nil||seen[n.id]{continue};seen[n.id]=true;if len(n.data)!=93||n.data[92]!=byte(n.id){panic("graph data")};sum+=n.id+1;queue[tail]=n.left;queue[tail+1]=n.right;tail+=2};return sum}
func build(){var nodes [96]*Node;for i:=range nodes{nodes[i]=&Node{id:i,data:make([]byte,93)};nodes[i].data[92]=byte(i)};for i:=range nodes{nodes[i].left=nodes[int(next()%96)];nodes[i].right=nodes[int(next()%96)]};root=nodes[0];expected=checksum()}
func waste(){b:=make([]byte,2000);b[1999]=99}
func main(){for round:=0;round<16;round++{build();for j:=0;j<160;j++{waste()};runtime.Collect();if checksum()!=expected{panic("graph reachability")};root=nil};println("PASS")}
`},
	{"large-live-segments", "16MiB", `package main
import "renvo.dev/x/runtime"
var blocks [32][]byte
func main(){for i:=range blocks{blocks[i]=make([]byte,262144);blocks[i][0]=byte(i);blocks[i][262143]=byte(i+1)};runtime.Collect();for i,b:=range blocks{if b[0]!=byte(i)||b[262143]!=byte(i+1){panic("segment")}};println("PASS")}
`},
	{"large-reverse-chain", "2MiB", `package main
import "renvo.dev/x/runtime"
type Node struct{next *Node; id int}
var root *Node
func build(){for i:=0;i<8192;i++{root=&Node{next:root,id:i}}}
func main(){build();runtime.Collect();p:=root;for i:=8191;i>=0;i--{if p==nil||p.id!=i{panic("chain")};p=p.next};if p!=nil{panic("tail")};println("PASS")}
`},
	{"reverse-chain", "128KiB", `package main
import "renvo.dev/x/runtime"
type Node struct{next *Node; id int; data [13]byte}
var root *Node
func build(){for i:=0;i<384;i++{root=&Node{next:root,id:i};root.data[12]=byte(i)}}
func waste(){b:=make([]byte,6000);b[5999]=99}
func main(){build();for round:=0;round<12;round++{for j:=0;j<20;j++{waste()};runtime.Collect();p:=root;for i:=383;i>=0;i--{if p==nil||p.id!=i||p.data[12]!=byte(i){panic("lost chain")};p=p.next};if p!=nil{panic("tail")}};println("PASS")}
`},
	{"recursive-stack-roots", "64KiB", `package main
import "renvo.dev/x/runtime"
func churn(){for i:=0;i<80;i++{b:=make([]byte,1300);b[1299]=byte(i)}}
func walk(depth int)int{b:=make([]byte,127);b[126]=byte(depth);if depth==0{churn();runtime.Collect()}else{if walk(depth-1)!=depth-1{panic("return")}};if b[126]!=byte(depth){panic("stack root")};return depth}
func main(){for i:=0;i<8;i++{if walk(48)!=48{panic("depth")}};println("PASS")}
`},
	{"defer-panic-roots", "64KiB", `package main
import "renvo.dev/x/runtime"
type Box struct{data []byte}
func churn(){for i:=0;i<100;i++{b:=make([]byte,777);b[776]=byte(i)}}
func attempt(){defer func(){value:=recover();runtime.Collect();churn();box,ok:=value.(*Box);if !ok||box.data[899]!=71{panic("panic root")}}();box:=&Box{data:make([]byte,900)};box.data[899]=71;panic(box)}
func deferred(b []byte){runtime.Collect();churn();if b[999]!=42{panic("defer root")}}
func setup(){b:=make([]byte,1000);b[999]=42;defer deferred(b);b=nil;churn()}
func main(){for i:=0;i<12;i++{attempt();setup()};println("PASS")}
`},
	{"interior-string-and-append", "64KiB", `package main
import "renvo.dev/x/runtime"
func text()string{b:=make([]byte,1800);for i:=range b{b[i]=byte(65+i%26)};return string(b)[1733:1799]}
func churn(){for i:=0;i<100;i++{b:=make([]byte,1100);b[1099]=2}}
func main(){s:=text();b:=make([]byte,17);b[16]=37;p:=&b[16];for i:=0;i<2000;i++{b=append(b,byte(i))};churn();runtime.Collect();if *p!=37||len(s)!=66||s[0]!=byte(65+1733%26)||s[65]!=byte(65+1798%26)||b[2016]!=byte(1999%256){panic("interior")};println("PASS")}
`},
	{"oom-recovery-repeat", "64KiB", `package main
import "renvo.dev/x/runtime"
var held []byte
var recovered int
var allocated bool
func attempt(){defer func(){if recover()==nil{panic("missing oom")};recovered++}();held=make([]byte,200000);allocated=true}
func churn(){b:=make([]byte,3000);b[2999]=55}
func main(){live:=make([]byte,4096);live[4095]=83;for i:=0;i<80;i++{attempt();churn();runtime.Collect();if live[4095]!=83{panic("lost after oom")}};if recovered!=80||allocated{panic("count")};println("PASS")}
`},
	{"global-initializer-pressure", "64KiB", `package main
import "renvo.dev/x/runtime"
type Box struct{b []byte}
var first=&Box{b:make([]byte,1024)}
var second=initialize()
func waste(){b:=make([]byte,3000);b[2999]=9}
func initialize()*Box{first.b[1023]=73;for i:=0;i<80;i++{waste()};runtime.Collect();return &Box{b:make([]byte,1500)}}
func main(){runtime.Collect();if first.b[1023]!=73||len(second.b)!=1500{panic("initializer")};println("PASS")}
`},
	{"mixed-size-reuse", "128KiB", `package main
import "renvo.dev/x/runtime"
var slots [24][]byte
var seed uint32=1234567
func next()uint32{seed=seed*1664525+1013904223;return seed}
func replace(index int,step int){size:=1+int(next()%1700);b:=make([]byte,size);for _,v:=range b{if v!=0{panic("not zero")}};for i:=range b{b[i]=byte(index+1)};slots[index]=b}
func verify(){for i,b:=range slots{for _,v:=range b{if v!=byte(i+1){panic("payload")}}}}
func main(){for step:=0;step<1600;step++{index:=int(next()%24);if step%4==0{slots[index]=nil}else{replace(index,step)};if step%19==0{runtime.Collect();verify()}};verify();println("PASS")}
`},
}
