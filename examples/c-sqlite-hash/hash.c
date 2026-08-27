#include "hash.h"
#include "arena.h"

static unsigned int fold(unsigned char ch) {
  if (ch >= 'A' && ch <= 'Z') return (unsigned int)(ch + ('a' - 'A'));
  return ch;
}

static unsigned int key_hash(const char *key, int length) {
  unsigned int value = 0;
  int i;
  for (i = 0; i < length; i++) value = (value << 3) ^ value ^ fold((unsigned char)key[i]);
  return value & 0x7fffffffU;
}

static int key_equal(const char *left, int left_length, const char *right, int right_length) {
  int i;
  if (left_length != right_length) return 0;
  for (i = 0; i < left_length; i++) {
    if (fold((unsigned char)left[i]) != fold((unsigned char)right[i])) return 0;
  }
  return 1;
}

static char *copy_key(const char *key, int length) {
  char *copy = (char *)arena_alloc((unsigned long)length + 1);
  int i;
  if (!copy) return 0;
  for (i = 0; i < length; i++) copy[i] = key[i];
  copy[length] = 0;
  return copy;
}

void hash_init(Hash *table) {
  table->buckets = 0;
  table->bucket_count = 0;
  table->count = 0;
  table->first = 0;
  table->last = 0;
}

static int resize(Hash *table, int size) {
  HashBucket *buckets = (HashBucket *)arena_alloc((unsigned long)size * sizeof(HashBucket));
  HashEntry *entry;
  int i;
  if (!buckets) return 0;
  for (i = 0; i < size; i++) {
    buckets[i].first = 0;
    buckets[i].count = 0;
  }
  for (entry = table->first; entry; entry = entry->list_next) {
    unsigned int bucket = key_hash(entry->key, entry->key_length) & (unsigned int)(size - 1);
    entry->bucket_next = buckets[bucket].first;
    buckets[bucket].first = entry;
    buckets[bucket].count++;
  }
  table->buckets = buckets;
  table->bucket_count = size;
  return 1;
}

static HashEntry *find_entry(const Hash *table, const char *key, int length, unsigned int *bucket_out) {
  unsigned int bucket;
  HashEntry *entry;
  if (!table->bucket_count) return 0;
  bucket = key_hash(key, length) & (unsigned int)(table->bucket_count - 1);
  if (bucket_out) *bucket_out = bucket;
  for (entry = table->buckets[bucket].first; entry; entry = entry->bucket_next) {
    if (key_equal(entry->key, entry->key_length, key, length)) return entry;
  }
  return 0;
}

void *hash_insert(Hash *table, const char *key, int length, void *value) {
  unsigned int bucket = 0;
  HashEntry *entry = find_entry(table, key, length, &bucket);
  void *previous;
  if (entry) {
    previous = entry->value;
    entry->value = value;
    return previous;
  }
  if (!table->bucket_count && !resize(table, 8)) return 0;
  if (table->count >= table->bucket_count * 2 && !resize(table, table->bucket_count * 2)) return 0;
  bucket = key_hash(key, length) & (unsigned int)(table->bucket_count - 1);
  entry = (HashEntry *)arena_alloc(sizeof(HashEntry));
  if (!entry) return 0;
  entry->key = copy_key(key, length);
  if (!entry->key) return 0;
  entry->key_length = length;
  entry->value = value;
  entry->bucket_next = table->buckets[bucket].first;
  entry->list_next = 0;
  table->buckets[bucket].first = entry;
  table->buckets[bucket].count++;
  if (table->last) table->last->list_next = entry;
  else table->first = entry;
  table->last = entry;
  table->count++;
  return 0;
}

void *hash_find(const Hash *table, const char *key, int length) {
  HashEntry *entry = find_entry(table, key, length, 0);
  return entry ? entry->value : 0;
}

void *hash_remove(Hash *table, const char *key, int length) {
  unsigned int bucket;
  HashEntry *entry = find_entry(table, key, length, &bucket);
  HashEntry *cursor;
  HashEntry *previous = 0;
  void *value;
  if (!entry) return 0;
  for (cursor = table->buckets[bucket].first; cursor != entry; cursor = cursor->bucket_next) previous = cursor;
  if (previous) previous->bucket_next = entry->bucket_next;
  else table->buckets[bucket].first = entry->bucket_next;
  previous = 0;
  for (cursor = table->first; cursor != entry; cursor = cursor->list_next) previous = cursor;
  if (previous) previous->list_next = entry->list_next;
  else table->first = entry->list_next;
  if (table->last == entry) table->last = previous;
  table->buckets[bucket].count--;
  table->count--;
  value = entry->value;
  return value;
}

int hash_count(const Hash *table) { return table->count; }
HashEntry *hash_first(const Hash *table) { return table->first; }
