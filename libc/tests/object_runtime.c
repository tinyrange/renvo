#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/resource.h>
#include <errno.h>
extern char *program_invocation_name;
extern char *program_invocation_short_name;
static void finish(void) { puts("PASS"); }
int main(int argc,char **argv) {
 struct rlimit r;
 if (argc!=2 || strcmp(argv[1],"argument")) return 1;
 if (strcmp(program_invocation_name,"/virtual/app") || strcmp(program_invocation_short_name,"app")) return 2;
 if (!getenv("EXAMPLE") || strcmp(getenv("EXAMPLE"),"value=tail") || getenv("MISSING")) return 3;
 if (getrlimit(RLIMIT_NOFILE,&r) || r.rlim_cur<16 || r.rlim_max<r.rlim_cur) return 4;
 errno=0; if (getrlimit(RLIMIT_NOFILE,0)!=-1 || errno!=EFAULT) return 5;
 char *p=malloc(32); if (!p) return 6;
 strcpy(p,"ababc"); if (strstr(p,"abc")!=p+2 || strstr(p,"")!=p || strstr(p,"abcd")) return 7;
 if (strnlen(p,3)!=3 || strnlen(p,20)!=5 || strnlen(p,0)!=0) return 8;
 free(p);
 if (atexit(finish)) return 9;
 return 0;
}
