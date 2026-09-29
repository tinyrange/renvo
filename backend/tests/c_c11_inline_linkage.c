extern int puts(const char *);
inline int header_add(int n) { return n+1; }
extern inline int exported_add(int n) { return n+2; }
inline int later_export(int n) { return n+3; }
extern int later_export(int);
extern inline __attribute__((gnu_inline)) int gnu_local(int n) { return n+4; }
int main(void) {
 if (header_add(1)!=2 || exported_add(1)!=3 || later_export(1)!=4 || gnu_local(1)!=5) return 1;
 puts("PASS");return 0;
}
