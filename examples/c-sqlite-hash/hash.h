#ifndef DEMO_HASH_H
#define DEMO_HASH_H

typedef struct HashEntry HashEntry;
typedef struct HashBucket HashBucket;
typedef struct Hash Hash;

struct HashEntry {
  HashEntry *bucket_next;
  HashEntry *list_next;
  const char *key;
  int key_length;
  void *value;
};

struct HashBucket {
  HashEntry *first;
  int count;
};

struct Hash {
  HashBucket *buckets;
  int bucket_count;
  int count;
  HashEntry *first;
  HashEntry *last;
};

void hash_init(Hash *table);
void *hash_insert(Hash *table, const char *key, int key_length, void *value);
void *hash_find(const Hash *table, const char *key, int key_length);
void *hash_remove(Hash *table, const char *key, int key_length);
int hash_count(const Hash *table);
HashEntry *hash_first(const Hash *table);

#endif
