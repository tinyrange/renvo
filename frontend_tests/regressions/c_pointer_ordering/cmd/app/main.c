#include <string.h>

int main(void) {
 char bytes[6] = "abcde";
 char *first = &bytes[0];
 char *last = &bytes[4];
 if (!(first < last && last > first && first <= first && last >= first)) return 1;
 memmove(bytes + 1, bytes, 4);
 if (bytes[0] != 'a' || bytes[1] != 'a' || bytes[4] != 'd') return 1;
 print("PASS\n");
 return 0;
}
