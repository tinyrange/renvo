#include "arena.h"

static unsigned char storage[32768];
static unsigned long used;

void arena_reset(void) {
  used = 0;
}

void *arena_alloc(unsigned long size) {
  unsigned long aligned = (size + 7) & ~7UL;
  void *result;
  if (aligned > sizeof(storage) - used) return 0;
  result = &storage[used];
  used += aligned;
  return result;
}
