extern int puts(const char *);
static int calls;
static int side(void) { ++calls; return 1; }
int main(void) {
 char buf[256];
 struct item { short x; char text[17]; } item;
 int n=0;
 _Static_assert(sizeof buf == 256, "array before decay");
 _Static_assert(sizeof item.text == 17, "member array");
 _Static_assert(sizeof buf[side()] == 1, "unevaluated index");
 _Static_assert(sizeof n++ == sizeof(int), "unevaluated increment");
 _Static_assert(sizeof *(&n) + 2 == sizeof(int) + 2, "unary binding");
 _Static_assert(sizeof side() == sizeof(int), "unevaluated call");
 if (n || calls) return 1;
 puts("PASS");return 0;
}
