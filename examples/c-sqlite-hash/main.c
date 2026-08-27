#include "hash.h"
#include "arena.h"

struct Record { int id; int score; };
static const char *keys[] = { "Ada", "Grace", "Edsger", "Barbara", "Ken", "Dennis", "Margaret", "Niklaus" };

static int text_length(const char *text) {
  int length = 0;
  while (text[length]) length++;
  return length;
}

int main(void) {
  struct Record records[8];
  struct Record replacement;
  Hash table;
  HashEntry *entry;
  int i;
  int total = 0;
  arena_reset();
  hash_init(&table);
  for (i = 0; i < 8; i++) {
    records[i].id = i + 1;
    records[i].score = (i + 3) * 7;
    hash_insert(&table, keys[i], text_length(keys[i]), &records[i]);
  }
  if (hash_count(&table) != 8) return 1;
  if (((struct Record *)hash_find(&table, "gRaCe", 5))->id != 2) return 2;
  replacement.id = 42; replacement.score = 100;
  if (hash_insert(&table, "Ada", 3, &replacement) != &records[0]) return 3;
  if (hash_remove(&table, "Dennis", 6) != &records[5]) return 4;
  for (entry = hash_first(&table); entry; entry = entry->list_next) total += ((struct Record *)entry->value)->score;
  return hash_count(&table) == 7 && total == 387 ? 0 : 5;
}
