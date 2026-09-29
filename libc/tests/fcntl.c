#include <fcntl.h>
#include <unistd.h>
#include <errno.h>
#include <stdio.h>
int main(void) {
 FILE *f=fopen("fcntl-data","w+"); if (!f) return 1;
 int fd=fileno(f), original=fcntl(fd,F_GETFL);
 if ((original&O_ACCMODE)!=O_RDWR || (original&(O_CREAT|O_TRUNC))) return 2;
 if (fcntl(fd,F_SETFD,FD_CLOEXEC) || fcntl(fd,F_GETFD)!=FD_CLOEXEC) return 3;
 int a=fcntl(fd,F_DUPFD,20); if(a!=20 || fcntl(a,F_GETFD)!=0) return 4;
 int b=fcntl(fd,F_DUPFD_CLOEXEC,20); if(b!=21 || fcntl(b,F_GETFD)!=FD_CLOEXEC) return 5;
 if(fcntl(fd,F_GETFD)!=FD_CLOEXEC) return 6;
 if(dup2(b,b)!=b || fcntl(b,F_GETFD)!=FD_CLOEXEC) return 7;
 if(dup2(fd,b)!=b || fcntl(b,F_GETFD)!=0) return 8;
 if(fcntl(a,F_SETFL,original|O_APPEND) || !(fcntl(fd,F_GETFL)&O_APPEND)) return 9;
 if(fputs("abc",f)<0) return 10;
 FILE *other=fopen("fcntl-data","a");if(!other || fputs("X",other)<0 || fclose(other))return 11;
 if(fputs("Y",f)<0 || fcntl(b,F_SETFL,original) || (fcntl(fd,F_GETFL)&O_APPEND))return 12;
 errno=0;if(fcntl(-1,F_GETFD)!=-1 || errno!=EBADF)return 13;
 errno=0;if(fcntl(fd,F_DUPFD,-1)!=-1 || errno!=EINVAL)return 14;
 if(close(a)||close(b)||fclose(f))return 15;
 f=fopen("fcntl-data","r");if(!f)return 16;
 char text[8];if(!fgets(text,8,f)||text[0]!='a'||text[3]!='X'||text[4]!='Y'||text[5]!=0||fclose(f))return 17;
 puts("PASS"); return 0;
}
