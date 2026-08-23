#include <stdio.h>
#include <stdint.h>

struct __renvo_FILE { int fd; };
static FILE __renvo_stdin = { 0 };
static FILE __renvo_stdout = { 1 };
static FILE __renvo_stderr = { 2 };
FILE *stdin = &__renvo_stdin;
FILE *stdout = &__renvo_stdout;
FILE *stderr = &__renvo_stderr;

extern int __renvo_c_write_byte(int fd, int ch);
extern int __renvo_c_read_byte(int fd);

int fputc(int ch, FILE *stream) { return __renvo_c_write_byte(stream->fd, ch); }
int putchar(int ch) { return fputc(ch, stdout); }
int getchar(void) { return __renvo_c_read_byte(stdin->fd); }

int fputs(const char *restrict text, FILE *restrict stream) {
    int count = 0;
    while (*text != '\0') { fputc((unsigned char)*text++, stream); count++; }
    return count;
}

size_t fwrite(const void *restrict ptr, size_t size, size_t count, FILE *restrict stream) {
    const unsigned char *bytes = ptr;
    size_t total = size * count;
    size_t i;
    for (i = 0; i < total; i++) if (fputc(bytes[i], stream) == EOF) return size == 0 ? 0 : i / size;
    return count;
}

size_t fread(void *restrict ptr, size_t size, size_t count, FILE *restrict stream) {
    unsigned char *bytes = ptr;
    size_t total = size * count;
    size_t i;
    for (i = 0; i < total; i++) {
        int ch = __renvo_c_read_byte(stream->fd);
        if (ch == EOF) return size == 0 ? 0 : i / size;
        bytes[i] = (unsigned char)ch;
    }
    return count;
}

int puts(const char *text) {
    int count = 0;
    while (*text != '\0') { putchar((unsigned char)*text++); count++; }
    putchar('\n');
    return count + 1;
}

static int __renvo_print_unsigned(FILE *stream, unsigned long long value, unsigned base, int upper, int width, int zero) {
    char digits[32];
    int used = 0;
    int count = 0;
    const char *alphabet = upper ? "0123456789ABCDEF" : "0123456789abcdef";
    do { digits[used++] = alphabet[value % base]; value /= base; } while (value != 0);
    while (width-- > used) { fputc(zero ? '0' : ' ', stream); count++; }
    while (used != 0) { fputc(digits[--used], stream); count++; }
    return count;
}

#if defined(__STDC_NO_IEC_60559_BFP__)
static int __renvo_print_float(FILE *stream, double value, int precision) {
    (void)value;
    (void)precision;
    return fputs("<float unavailable>", stream);
}
#else
static int __renvo_print_float(FILE *stream, double value, int precision) {
    unsigned long long whole;
    double fraction;
    int count = 0;
    int i;
    if (value != value) { fputs("nan", stream); return 3; }
    if (value < 0) { fputc('-', stream); count++; value = -value; }
    whole = (unsigned long long)value;
    fraction = value - (double)whole;
    count += __renvo_print_unsigned(stream, whole, 10, 0, 0, 0);
    if (precision > 0) {
        fputc('.', stream); count++;
        for (i = 0; i < precision; i++) {
            int digit;
            fraction *= 10.0;
            digit = (int)fraction;
            fputc('0' + digit, stream);
            count++;
            fraction -= digit;
        }
    }
    return count;
}
#endif

int vfprintf(FILE *restrict stream, const char *restrict format, va_list args) {
    int count = 0;
    while (*format != '\0') {
        int width = 0;
        int zero = 0;
        int long_count = 0;
        int precision = -1;
        char spec;
        if (*format != '%') { fputc((unsigned char)*format++, stream); count++; continue; }
        format++;
        if (*format == '%') { fputc('%', stream); format++; count++; continue; }
        if (*format == '0') { zero = 1; format++; }
        while (*format >= '0' && *format <= '9') { width = width * 10 + *format++ - '0'; }
        if (*format == '.') {
            precision = 0;
            format++;
            while (*format >= '0' && *format <= '9') { precision = precision * 10 + *format++ - '0'; }
        }
        while (*format == 'l') { long_count++; format++; }
        spec = *format++;
        if (spec == 's') {
            const char *text = va_arg(args, const char *);
            if (text == NULL) text = "(null)";
            while (*text != '\0') { fputc((unsigned char)*text++, stream); count++; }
        } else if (spec == 'c') {
            fputc(va_arg(args, int), stream); count++;
        } else if (spec == 'd' || spec == 'i') {
            long long value = long_count > 1 ? va_arg(args, long long) : long_count ? va_arg(args, long) : va_arg(args, int);
            unsigned long long magnitude;
            if (value < 0) { fputc('-', stream); count++; magnitude = (unsigned long long)(-(value + 1)) + 1; }
            else magnitude = (unsigned long long)value;
            count += __renvo_print_unsigned(stream, magnitude, 10, 0, width, zero);
        } else if (spec == 'u' || spec == 'x' || spec == 'X' || spec == 'o') {
            unsigned long long value = long_count > 1 ? va_arg(args, unsigned long long) : long_count ? va_arg(args, unsigned long) : va_arg(args, unsigned int);
            unsigned base = spec == 'o' ? 8 : (spec == 'u' ? 10 : 16);
            count += __renvo_print_unsigned(stream, value, base, spec == 'X', width, zero);
        } else if (spec == 'p') {
            uintptr_t value = (uintptr_t)va_arg(args, void *);
            fputc('0', stream); fputc('x', stream); count += 2 + __renvo_print_unsigned(stream, value, 16, 0, (int)(2 * sizeof(void *)), 1);
        } else if (spec == 'f' || spec == 'F') {
            if (precision < 0) precision = 6;
            count += __renvo_print_float(stream, va_arg(args, double), precision);
        } else {
            fputc('%', stream); fputc(spec, stream); count += 2;
        }
    }
    return count;
}

int vprintf(const char *restrict format, va_list args) { return vfprintf(stdout, format, args); }

int printf(const char *restrict format, ...) {
    int result;
    va_list args;
    va_start(args, format);
    result = vprintf(format, args);
    va_end(args);
    return result;
}

int fprintf(FILE *restrict stream, const char *restrict format, ...) {
    int result;
    va_list args;
    va_start(args, format);
    result = vfprintf(stream, format, args);
    va_end(args);
    return result;
}
