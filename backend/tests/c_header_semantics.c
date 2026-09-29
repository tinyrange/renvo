#include <stdio.h>
#include <stddef.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <time.h>
#include <stdlib.h>
#include <string.h>
#include <locale.h>
#include <limits.h>
#include <errno.h>
#include <unistd.h>
#include <stdarg.h>
#include <stdnoreturn.h>

_Static_assert(sizeof(size_t) == 8, "size" " type");
_Static_assert(sizeof(off_t) == 8, "offset");
_Static_assert(sizeof(struct timespec) == 16, "timespec");
_Static_assert(sizeof(struct stat) == 144, "stat");
_Static_assert(offsetof(struct stat, st_mode) == 24, "mode");
_Static_assert(offsetof(struct stat, st_size) == 48, "size");
_Static_assert(offsetof(struct stat, st_atim) == 72, "atime");
_Static_assert(offsetof(struct stat, st_ctim) == 104, "ctime");

noreturn void terminate_test(int status);
void terminate_test(int status) { exit(status); }
static int format(char *b, size_t n, const char *fmt, ...) {
    va_list args;
    va_start(args, fmt);
    int result = vsnprintf(b, n, fmt, args);
    va_end(args);
    return result;
}
int main(int argc, char **argv) {
    const char *message;
    message = ((const char *)("PASS"));
    if (argc < 1 || strcmp(program_invocation_name, argv[0])) return 1;
    char *short_name = strrchr(argv[0], '/');
    if (strcmp(program_invocation_short_name, short_name ? short_name + 1 : argv[0])) return 2;
    char b[32];
    memset(b, '!', sizeof(b));
    if (snprintf(b, 5, "%s:%d", "abcdef", 42) != 9) return 3;
    if (strcmp(b, "abcd") || b[5] != '!') return 4;
    if (snprintf(NULL, 0, "%s:%d", "abcdef", 42) != 9) return 5;
    if (format(b, 1, "%u", 123u) != 3 || b[0] != 0 || b[1] != 'b') return 6;
    if (sprintf(b, "%s:%d:%x", "abc", -42, 255u) != 10 || strcmp(b, "abc:-42:ff")) return 7;
    if (format(b, sizeof(b), "%s:%ld", "x", 123456789L) != 11 || strcmp(b, "x:123456789")) return 8;
    char *copy = strdup("hello");
    if (!copy || strcmp(copy, "hello")) return 9;
    copy[0] = 'H'; free(copy);
    copy = strndup("hello", 3);
    if (!copy || strcmp(copy, "hel")) return 10;
    free(copy);
    copy = strndup("hello", 0);
    if (!copy || copy[0]) return 11;
    free(copy);
    if (MB_CUR_MAX != 1 || MB_LEN_MAX < 4) return 12;
    if (!setlocale(LC_CTYPE, "C.UTF-8") || MB_CUR_MAX != 4) return 13;
    if (!setlocale(LC_NUMERIC, "C") || MB_CUR_MAX != 4) return 14;
    if (setlocale(LC_ALL, "invalid") || MB_CUR_MAX != 4) return 15;
    if (!setlocale(LC_ALL, "C") || MB_CUR_MAX != 1) return 16;
    errno = 0;
    if (dup2(-1, 8) != -1 || errno != EBADF) return 17;
    if (dup2(1, 1) != 1 || dup2(1, 8) != 8) return 18;
    if (close(1) || dup2(8, 1) != 1 || close(8)) return 19;
    if (fputs(message, stdout) < 0) return 20;
    if (putchar('\n') == EOF) return 20;
    return 0;
}
