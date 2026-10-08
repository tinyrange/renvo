/* Original freestanding integration port. Upstream algorithms are unchanged.
 * Only Linux write(64), clock_gettime(113), and exit(93) are used.
 * The startup runs both required seed sets; each has its own timed interval.
 */
#include "coremark.h"
#include <stdarg.h>
volatile ee_s32 seed1_volatile = 0;
volatile ee_s32 seed2_volatile = 0;
volatile ee_s32 seed3_volatile = 0x66;
volatile ee_s32 seed4_volatile = ITERATIONS;
volatile ee_s32 seed5_volatile = 0;
ee_u32 default_num_contexts = 1;
static unsigned long start_ticks, stop_ticks;
static long syscall3(long number, long a, long b, long c) {
 register long x8 __asm__("x8") = number;
 register long x0 __asm__("x0") = a;
 register long x1 __asm__("x1") = b;
 register long x2 __asm__("x2") = c;
 __asm__ volatile("svc #0" : "+r"(x0) : "r"(x8), "r"(x1), "r"(x2) : "memory", "cc");
 return x0;
}
static void fatal(void) {
 syscall3(93, 125, 0, 0);
 for (;;) {}
}
static unsigned long ticks(void) {
 struct { long sec, nsec; } ts;
 if (syscall3(113, 1, (long)&ts, 0) != 0) fatal();
 return (unsigned long)ts.sec * 1000000ul + (unsigned long)ts.nsec / 1000ul;
}
void start_time(void) { start_ticks = ticks(); }
void stop_time(void) { stop_ticks = ticks(); }
CORE_TICKS get_time(void) { return stop_ticks - start_ticks; }
secs_ret time_in_secs(CORE_TICKS elapsed) { return (secs_ret)(elapsed / 1000000ul); }
void portable_init(core_portable *p, int *argc, char *argv[]) {
 (void)argc; (void)argv;
 if (sizeof(ee_ptr_int) != sizeof(void *) || sizeof(ee_u32) != 4 || sizeof(ee_u16) != 2) fatal();
 p->portable_id = 1;
}
void portable_fini(core_portable *p) { p->portable_id = 0; }
void select_validation_seeds(void) { seed1_volatile = 0x3415; seed2_volatile = 0x3415; }
void *memcpy(void *destination, const void *source, ee_size_t n) {
 unsigned char *d = destination; const unsigned char *s = source;
 for (ee_size_t i = 0; i < n; i++) d[i] = s[i];
 return destination;
}
void *memset(void *destination, int value, ee_size_t n) {
 unsigned char *d = destination;
 for (ee_size_t i = 0; i < n; i++) d[i] = (unsigned char)value;
 return destination;
}
static char output[1024];
static unsigned int used;
static void flush(void) {
 unsigned int sent = 0;
 while (sent < used) {
  long n = syscall3(64, 1, (long)(output+sent), used-sent);
  if (n == -4) continue;
  if (n <= 0 || (unsigned long)n > used-sent) fatal();
  sent += (unsigned int)n;
 }
 used = 0;
}
static void emit(char c) {
 if (used == sizeof(output)) flush();
 output[used++] = c;
}
int ee_printf(const char *format, ...) {
 va_list args; va_start(args, format);
 int count = 0;
 while (*format) {
  if (*format != '%') { emit(*format++); count++; continue; }
  format++;
  if (*format == '%') { emit(*format++); count++; continue; }
  char pad = ' ';
  if (*format == '0') { pad = '0'; format++; }
  unsigned int width = 0;
  while (*format >= '0' && *format <= '9') { width = width*10 + (*format++ - '0'); }
  if (width > 64) fatal();
  int wide = 0;
  if (*format == 'l') { wide = 1; format++; }
  char spec = *format++;
  if (spec == 's') {
   const char *s = va_arg(args, const char *);
   while (*s) { emit(*s++); count++; }
   continue;
  }
  unsigned long value;
  int negative = 0;
  if (spec == 'd') {
   long signed_value = wide ? va_arg(args, long) : va_arg(args, int);
   negative = signed_value < 0;
   value = negative ? 0ul-(unsigned long)signed_value : (unsigned long)signed_value;
  } else if (spec == 'u' || spec == 'x') {
   value = wide ? va_arg(args, unsigned long) : va_arg(args, unsigned int);
  } else { fatal(); value = 0; }
  unsigned int base = spec == 'x' ? 16 : 10;
  char digits[32]; unsigned int n = 0;
  do { digits[n++] = "0123456789abcdef"[value % base]; value /= base; } while (value);
  if (negative) { emit('-'); count++; if (width) width--; }
  while (width > n) { emit(pad); count++; width--; }
  while (n) { emit(digits[--n]); count++; }
 }
 va_end(args);
 flush();
 return count;
}
