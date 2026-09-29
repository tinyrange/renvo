/* Linux/amd64 bridge for separately linked libc objects. */
#include <stdint.h>
#include <stdlib.h>
extern long __renvo_linux_syscall(long, uintptr_t, uintptr_t, uintptr_t);
static char **__renvo_environ;
char *program_invocation_name;
char *program_invocation_short_name;
extern int main(int, char **);
int __renvo_libc_start(int argc, char **argv) {
 __renvo_environ = argv + argc + 1;
 program_invocation_name = argc ? argv[0] : "";
 program_invocation_short_name = program_invocation_name;
 for (char *p=program_invocation_name; *p; p++) if (*p=='/') program_invocation_short_name=p+1;
 exit(main(argc,argv));
 return 0;
}
char *__renvo_c_getenv(const char *name) {
 if (!__renvo_environ || !name || !*name) return NULL;
 for (char **p=__renvo_environ; *p; p++) {
  unsigned long i=0;
  while (name[i] && name[i]!='=' && name[i]==(*p)[i]) i++;
  if (!name[i] && (*p)[i]=='=') return *p+i+1;
 }
 return NULL;
}
void __renvo_c_abort(int status) { __renvo_linux_syscall(60,status,0,0); for (;;) {} }
int __renvo_c_close(int fd) { return __renvo_linux_syscall(3,fd,0,0); }
int __renvo_c_dup2(int oldfd,int newfd) { return __renvo_linux_syscall(33,oldfd,newfd,0); }
int __renvo_c_open(const char *name,int flags) {
 int native=(flags&3)==3?2:((flags&2)?1:0);
 if(flags&4) native|=64;
 if(flags&8) native|=512;
 if(flags&16) native|=1024;
 if(flags&32) native|=128;
 return __renvo_linux_syscall(2,(uintptr_t)name,native,0666);
}
int __renvo_c_write_byte(int fd,int ch) {
 unsigned char b=ch;
 long n=__renvo_linux_syscall(1,fd,(uintptr_t)&b,1);
 return n<0?n:(n==1?b:-5);
}
int __renvo_c_read_byte(int fd,unsigned char *out) { return __renvo_linux_syscall(0,fd,(uintptr_t)out,1); }
