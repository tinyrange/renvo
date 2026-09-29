#include <wchar.h>
#include <errno.h>
#include <locale.h>
#include <string.h>

extern void __renvo_stream_error(FILE *stream, int code);
static mbstate_t __renvo_read_state;
static mbstate_t __renvo_write_state;
int mbsinit(const mbstate_t *state) { return state == NULL || state->remaining == 0; }
size_t mbrtowc(wchar_t *restrict out, const char *restrict text, size_t size, mbstate_t *restrict state) {
    size_t used = 0;
    unsigned int ch;
    if (state == NULL) state = &__renvo_read_state;
    if (text == NULL) { text = ""; size = 1; out = NULL; }
    if (size == 0) return (size_t)-2;
    if (!__renvo_locale_utf8()) {
        ch = (unsigned char)text[0];
        if (ch >= 128) { errno = EILSEQ; return (size_t)-1; }
        if (out != NULL) *out = (wchar_t)ch;
        state->remaining = 0;
        return ch == 0 ? 0 : 1;
    }
    if (state->remaining == 0) {
        ch = (unsigned char)text[used++];
        if (ch < 128) { if (out != NULL) *out = (wchar_t)ch; return ch == 0 ? 0 : 1; }
        if (ch >= 0xc2 && ch <= 0xdf) { state->value = ch & 31; state->minimum = 0x80; state->remaining = 1; }
        else if (ch >= 0xe0 && ch <= 0xef) { state->value = ch & 15; state->minimum = 0x800; state->remaining = 2; }
        else if (ch >= 0xf0 && ch <= 0xf4) { state->value = ch & 7; state->minimum = 0x10000; state->remaining = 3; }
        else { errno = EILSEQ; return (size_t)-1; }
    }
    while (state->remaining != 0) {
        if (used == size) return (size_t)-2;
        ch = (unsigned char)text[used++];
        if (ch < 0x80 || ch > 0xbf) { state->remaining = 0; errno = EILSEQ; return (size_t)-1; }
        state->value = (state->value << 6) | (ch & 63);
        state->remaining--;
    }
    ch = state->value;
    if (ch < state->minimum || ch > 0x10ffff || (ch >= 0xd800 && ch <= 0xdfff)) { errno = EILSEQ; return (size_t)-1; }
    if (out != NULL) *out = (wchar_t)ch;
    return used;
}
size_t mbrlen(const char *restrict text, size_t size, mbstate_t *restrict state) { return mbrtowc(NULL, text, size, state); }
size_t wcrtomb(char *restrict out, wchar_t wide, mbstate_t *restrict state) {
    unsigned int ch = (unsigned int)wide;
    if (state == NULL) state = &__renvo_write_state;
    state->remaining = 0;
    if (out == NULL) return 1;
    if (ch < 128) { out[0] = (char)ch; return 1; }
    if (!__renvo_locale_utf8() || ch > 0x10ffff || (ch >= 0xd800 && ch <= 0xdfff)) { errno = EILSEQ; return (size_t)-1; }
    if (ch < 0x800) { out[0] = (char)(0xc0 | (ch >> 6)); out[1] = (char)(0x80 | (ch & 63)); return 2; }
    if (ch < 0x10000) {
        out[0] = (char)(0xe0 | (ch >> 12)); out[1] = (char)(0x80 | ((ch >> 6) & 63)); out[2] = (char)(0x80 | (ch & 63)); return 3;
    }
    out[0] = (char)(0xf0 | (ch >> 18)); out[1] = (char)(0x80 | ((ch >> 12) & 63)); out[2] = (char)(0x80 | ((ch >> 6) & 63)); out[3] = (char)(0x80 | (ch & 63)); return 4;
}
size_t mbsrtowcs(wchar_t *restrict out, const char **restrict text, size_t size, mbstate_t *restrict state) {
    const char *cursor = *text;
    size_t count = 0, n;
    wchar_t ch;
    mbstate_t query;
    if (state == NULL) state = &__renvo_read_state;
    if (out == NULL) { query = *state; state = &query; }
    while (out == NULL || count < size) {
        /* Bound each decode at the terminating NUL, never read past it. */
        size_t available = 0;
        while (available < 4 && cursor[available] != '\0') available++;
        if (available < 4) available++;
        n = mbrtowc(&ch, cursor, available, state);
        if (n == (size_t)-1 || n == (size_t)-2) { if (out != NULL) *text = cursor; errno = EILSEQ; return (size_t)-1; }
        if (ch == 0) { if (out != NULL) { out[count] = 0; *text = NULL; } return count; }
        if (out != NULL) out[count] = ch;
        count++; cursor += n;
    }
    *text = cursor;
    return count;
}
size_t wcsrtombs(char *restrict out, const wchar_t **restrict text, size_t size, mbstate_t *restrict state) {
    const wchar_t *cursor = *text;
    size_t count = 0, n;
    char bytes[4];
    mbstate_t local;
    if (state == NULL) state = &__renvo_write_state;
    while (out == NULL || count < size) {
        local = *state;
        n = wcrtomb(bytes, *cursor, &local);
        if (n == (size_t)-1) { if (out != NULL) *text = cursor; return (size_t)-1; }
        if (out != NULL && n > size - count) break;
        if (out != NULL) { memcpy(out + count, bytes, n); *state = local; }
        if (*cursor == 0) { if (out != NULL) *text = NULL; return count; }
        count += n; cursor++;
    }
    if (out != NULL) *text = cursor;
    return count;
}
size_t wcslen(const wchar_t *text) { size_t n = 0; while (text[n]) n++; return n; }
int wcscmp(const wchar_t *left, const wchar_t *right) {
    while (*left && *left == *right) { left++; right++; }
    return *left < *right ? -1 : (*left > *right);
}
wchar_t *wcscpy(wchar_t *restrict out, const wchar_t *restrict text) { wchar_t *result = out; while ((*out++ = *text++) != 0) {} return result; }
wint_t fputwc(wchar_t ch, FILE *stream) {
    char data[4]; size_t n, i; mbstate_t state = {0};
    n = wcrtomb(data, ch, &state);
    if (n == (size_t)-1) { __renvo_stream_error(stream, errno); return WEOF; }
    for (i = 0; i < n; i++) if (fputc((unsigned char)data[i], stream) == EOF) return WEOF;
    return (wint_t)ch;
}
wint_t putwc(wchar_t ch, FILE *stream) { return fputwc(ch, stream); }
wint_t putwchar(wchar_t ch) { return fputwc(ch, stdout); }
int fputws(const wchar_t *restrict text, FILE *restrict stream) {
    int count = 0;
    while (*text) { if (fputwc(*text++, stream) == WEOF) return -1; count++; }
    return count;
}
/* Initial wide-format surface. Unknown conversions fail, never print a fake
   successful representation. Strings/chars and integer conversions are enough
   for the greeting path; width/precision/float remain explicitly unsupported. */
int vfwprintf(FILE *restrict stream, const wchar_t *restrict format, va_list args) {
    int count = 0, n, longs;
    wchar_t spec;
    while (*format) {
        if (*format != '%') { if (fputwc(*format++, stream) == WEOF) return -1; count++; continue; }
        format++; longs = 0;
        while (*format == 'l') { longs++; format++; }
        spec = *format++;
        if (spec == '%') { if (fputwc('%', stream) == WEOF) return -1; count++; }
        else if (spec == 's' && longs == 1) {
            const wchar_t *text = va_arg(args, const wchar_t *);
            n = fputws(text, stream); if (n < 0) return -1; count += n;
        } else if (spec == 's' && longs == 0) {
            const char *text = va_arg(args, const char *);
            wchar_t ch; mbstate_t state = {0}; size_t used;
            while (*text) {
                used = mbrtowc(&ch, text, strlen(text), &state);
                if (used == (size_t)-1 || used == (size_t)-2 || fputwc(ch, stream) == WEOF) return -1;
                text += used; count++;
            }
        } else if (spec == 'c' && longs <= 1) {
            int ch = va_arg(args, int);
            if (longs == 0 && (unsigned int)ch > 127) { errno = EILSEQ; return -1; }
            if (fputwc((wchar_t)ch, stream) == WEOF) return -1; count++;
        } else if ((spec == 'd' || spec == 'i') && longs <= 2) {
            long long value = longs == 2 ? va_arg(args, long long) : longs ? va_arg(args, long) : va_arg(args, int);
            n = fprintf(stream, "%lld", value); if (n < 0) return -1; count += n;
        } else if ((spec == 'u' || spec == 'x' || spec == 'o') && longs <= 2) {
            unsigned long long value = longs == 2 ? va_arg(args, unsigned long long) : longs ? va_arg(args, unsigned long) : va_arg(args, unsigned int);
            n = fprintf(stream, spec == 'u' ? "%llu" : spec == 'x' ? "%llx" : "%llo", value); if (n < 0) return -1; count += n;
        } else { errno = EINVAL; return -1; }
    }
    return count;
}
int fwprintf(FILE *restrict stream, const wchar_t *restrict format, ...) { int result; va_list args; va_start(args, format); result = vfwprintf(stream, format, args); va_end(args); return result; }
int vwprintf(const wchar_t *restrict format, va_list args) { return vfwprintf(stdout, format, args); }
int wprintf(const wchar_t *restrict format, ...) { int result; va_list args; va_start(args, format); result = vwprintf(format, args); va_end(args); return result; }
