/* Deliberately no headers: resolve libc from ELF undefined symbols. */
extern int fcntl(int, int, ...);
extern int errno;
int main(void) {
 if (fcntl(-1, 1) != -1 || errno != 9) return 1;
 int fd = fcntl(1, 1030, 10);
 if (fd < 10 || fcntl(fd, 1) != 1) return 2;
 if (fcntl(1, 1) != 0) return 3;
 return 0;
}
