/* Freestanding Linux/PPC64 ELFv2 init. Runs in the guest, not a host shim. */
static long call3(long n,long a,long b,long c){register long r0 __asm__("r0")=n;register long r3 __asm__("r3")=a;register long r4 __asm__("r4")=b;register long r5 __asm__("r5")=c;__asm__ volatile("sc" : "+r"(r3) : "r"(r0),"r"(r4),"r"(r5) : "r6","r7","r8","r9","r10","r11","r12","cr0","memory");return r3;}
static void text(const char *p){long n=0;while(p[n])n++;call3(4,1,(long)p,n);}
void smoke(void){char names[6][65];text("RENVO-POWER-LINUX: INIT\n");if(call3(122,(long)names,0,0)!=0){text("RENVO-POWER-LINUX: UNAME FAIL\n");for(;;){}};text(names[0]);text(" ");text(names[2]);text(" ");text(names[4]);text("\nRENVO-POWER-LINUX: PASS\n");call3(88,0xfee1dead,672274793,0x4321fedc);for(;;){}}
