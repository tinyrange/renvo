#include <stddef.h>
#include <stdio.h>

struct uf_node {
	struct uf_node *parent;
	unsigned int rank;
};

_Static_assert(sizeof(struct uf_node) == 16, "x86_64 uf_node size");
_Static_assert(offsetof(struct uf_node, rank) == 8, "x86_64 uf_node rank offset");

extern struct uf_node *uf_find(struct uf_node *node);
extern void uf_union(struct uf_node *node1, struct uf_node *node2);

int main(void)
{
	struct uf_node a = { &a, 0 };
	struct uf_node b = { &b, 0 };
	struct uf_node c = { &c, 0 };

	uf_union(&a, &b);
	if (uf_find(&a) != &a || uf_find(&b) != &a || a.rank != 1)
		return 1;
	uf_union(&c, &b);
	if (uf_find(&c) != &a || c.parent != &a || a.rank != 1)
		return 2;
	uf_union(&a, &c);
	if (uf_find(&a) != &a || a.rank != 1)
		return 3;
	puts("PASS");
	return 0;
}
