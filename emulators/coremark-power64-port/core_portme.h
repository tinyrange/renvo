/* Freestanding Linux PPC64le port for unmodified EEMBC CoreMark.
 * Retain the pinned upstream LICENSE.md and source copyright notices.
 * This port is original integration code, not part of upstream CoreMark.
 */
#ifndef CORE_PORTME_H
#define CORE_PORTME_H
#define HAS_FLOAT 0
#define HAS_TIME_H 0
#define USE_CLOCK 0
#define HAS_STDIO 0
#define HAS_PRINTF 0
#define SEED_METHOD SEED_VOLATILE
#define MEM_METHOD MEM_STACK
#define MULTITHREAD 1
#define MAIN_HAS_NOARGC 1
#define MAIN_HAS_NORETURN 0
#define COMPILER_VERSION "GCC " __VERSION__
#define COMPILER_FLAGS "-O2 -static -nostdlib -fno-builtin -fno-stack-protector -fno-pie -no-pie -mcpu=power8 -mno-altivec -mno-vsx -msoft-float -fno-tree-vectorize -I. -Iport -DITERATIONS=" CM_STRING(ITERATIONS) " -DPERFORMANCE_RUN=1 -Wl,-e,_start -Wl,--build-id=none"
#define CM_STRING_INNER(x) #x
#define CM_STRING(x) CM_STRING_INNER(x)
#define MEM_LOCATION "Stack; 2000 bytes; single context; freestanding Linux PPC64le"
typedef signed short ee_s16;
typedef unsigned short ee_u16;
typedef signed int ee_s32;
typedef unsigned int ee_u32;
typedef unsigned char ee_u8;
typedef unsigned long ee_ptr_int;
typedef unsigned long ee_size_t;
typedef double ee_f32;
#define NULL ((void *)0)
#define align_mem(x) ((void *)(4 + (((ee_ptr_int)(x)-1) & ~(ee_ptr_int)3)))
typedef unsigned long CORE_TICKS;
#define CORETIMETYPE CORE_TICKS
extern ee_u32 default_num_contexts;
typedef struct CORE_PORTABLE_S { ee_u8 portable_id; } core_portable;
void portable_init(core_portable *, int *, char *[]);
void portable_fini(core_portable *);
void run_guest(void);
int ee_printf(const char *, ...);
#endif
