CC := renvo cc
CFLAGS := -t linux/amd64 -nostdinc -I.

.PHONY: all
all: app

app: main.o hash.o arena.o
	$(CC) -t linux/amd64 $^ -o $@

main.o: main.c hash.h arena.h
	$(CC) $(CFLAGS) -c $< -o $@

hash.o: hash.c hash.h arena.h
	$(CC) $(CFLAGS) -c $< -o $@

arena.o: arena.c arena.h
	$(CC) $(CFLAGS) -c $< -o $@
