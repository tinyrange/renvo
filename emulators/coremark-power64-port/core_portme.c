/* Original freestanding integration port. Upstream algorithms are unchanged.
 * Linux PPC64 write(4), clock_gettime(246), uname(122), and reboot(88).
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
 register long r0 __asm__("r0") = number;
 register long r3 __asm__("r3") = a;
 register long r4 __asm__("r4") = b;
 register long r5 __asm__("r5") = c;
 /* PPC Linux signals errno using CR0.SO and a positive r3. */
 __asm__ volatile("sc\n\tbns 1f\n\tneg %0,%0\n1:"
   : "+r"(r3), "+r"(r0), "+r"(r4), "+r"(r5)
   : : "r6", "r7", "r8", "r9", "r10", "r11", "r12", "cr0", "memory");
 return r3;
}
static void poweroff(void) {
 syscall3(88, 0xfee1dead, 672274793, 0x4321fedc);
 for (;;) {}
}
static void fatal(void) {
 static const char message[] = "RENVO-POWER-COREMARK: FAIL (port/syscall)\n";
 syscall3(4, 1, (long)message, sizeof(message)-1);
 poweroff();
}
static unsigned int validated_runs;
static int same(const char *a, const char *b) {
 while (*a && *a == *b) { a++; b++; }
 return *a == *b;
}
static unsigned long ticks(void) {
 struct { long sec, nsec; } ts;
 if (syscall3(246, 1, (long)&ts, 0) != 0) fatal();
 return (unsigned long)ts.sec * 1000000ul + (unsigned long)ts.nsec / 1000ul;
}
void start_time(void) { ee_printf("RENVO-POWER-COREMARK: START\n"); start_ticks = ticks(); }
void stop_time(void) { stop_ticks = ticks(); ee_printf("RENVO-POWER-COREMARK: STOP\n"); }
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
  long n = syscall3(4, 1, (long)(output+sent), used-sent);
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
 if (same(format, "Correct operation validated. See README.md for run and reporting rules.\n")) validated_runs++;
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

int main(void);
void run_guest(void) {
 char names[6][65];
 ee_printf("RENVO-POWER-COREMARK: INIT\n");
 if (syscall3(122, (long)names, 0, 0) != 0) fatal();
 ee_printf("Guest: %s %s %s\n", names[0], names[2], names[4]);
 main();
 select_validation_seeds();
 main();
 if (validated_runs == 2)
   ee_printf("RENVO-POWER-COREMARK: PASS\n");
 else
   ee_printf("RENVO-POWER-COREMARK: FAIL (upstream validation/duration)\n");
 poweroff();
}
