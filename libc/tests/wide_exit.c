#include <wchar.h>
#include <locale.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>
#define CHECK(x) do { if (!(x)) { fprintf(stderr, "failed line %d errno=%d\n", __LINE__, errno); _Exit(1); } } while (0)
static void first(void) { fputs("first\n", stderr); }
static void second(void) { fputs("second\n", stderr); }
int main(int argc, char **argv) {
    const char *text, *original;
    const wchar_t *wide;
    wchar_t buffer[32], ch;
    char bytes[32];
    mbstate_t state = {0};
    CHECK(argc >= 1);
    CHECK(getenv("RENVO_LIBC_TEST") != NULL);
    CHECK(strcmp(getenv("RENVO_LIBC_TEST"), "guest") == 0);
    CHECK(getenv("RENVO_MISSING_TEST_VALUE") == NULL);
    CHECK(getenv("") == NULL && getenv("RENVO_LIBC_TEST=") == NULL);
    CHECK(strcmp(setlocale(LC_ALL, NULL), "C") == 0);
    CHECK(setlocale(LC_ALL, "") != NULL);
    CHECK(strcmp(setlocale(LC_CTYPE, NULL), "C.UTF-8") == 0);
    CHECK(setlocale(LC_ALL, "not-installed") == NULL);
    CHECK(strcmp(setlocale(LC_CTYPE, NULL), "C.UTF-8") == 0);
    CHECK(mbrtowc(&ch, "\xe2", 1, &state) == (size_t)-2 && !mbsinit(&state));
    CHECK(mbrtowc(&ch, "\x82\xac", 2, &state) == 2 && ch == 0x20ac && mbsinit(&state));
    CHECK(wcrtomb(bytes, ch, &state) == 3 && memcmp(bytes, "\xe2\x82\xac", 3) == 0);
    CHECK(mbrtowc(&ch, "\xed\xa0\x80", 3, &state) == (size_t)-1 && errno == EILSEQ);
    CHECK(mbrtowc(&ch, "\xf4\x90\x80\x80", 4, &state) == (size_t)-1 && errno == EILSEQ);
    CHECK(mbrtowc(&ch, "\xc0\x80", 2, &state) == (size_t)-1 && errno == EILSEQ);
    text = "A\xc3\xa9\xf0\x9f\x98\x80"; original = text;
    CHECK(mbsrtowcs(NULL, &text, 0, &state) == 3 && text == original);
    CHECK(mbsrtowcs(buffer, &text, 1, &state) == 1 && text == original + 1);
    CHECK(mbsrtowcs(buffer + 1, &text, 31, &state) == 2 && text == NULL);
    CHECK(buffer[0] == 'A' && buffer[1] == 0xe9 && buffer[2] == 0x1f600 && buffer[3] == 0);
    wide = buffer;
    CHECK(wcsrtombs(NULL, &wide, 0, &state) == 7 && wide == buffer);
    CHECK(wcsrtombs(bytes, &wide, 2, &state) == 1 && wide == buffer + 1);
    CHECK(wcsrtombs(bytes + 1, &wide, 31, &state) == 6 && wide == NULL && strcmp(bytes, original) == 0);
    CHECK(wprintf(L"wide: %ls\n", buffer) == 10);
    CHECK(setlocale(LC_CTYPE, "C") != NULL);
    CHECK(wcrtomb(bytes, 0x20ac, &state) == (size_t)-1 && errno == EILSEQ);
    CHECK(atexit(first) == 0 && atexit(second) == 0);
    if (argc > 1 && strcmp(argv[1], "immediate") == 0) _Exit(0);
    if (argc > 1 && strcmp(argv[1], "exit") == 0) exit(0);
    return 0;
}
