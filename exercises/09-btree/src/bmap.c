#include <bmap.h>
#include <fs_malloc.h>

struct bmap
{
	u32 len;
	u64 data[0];
};

struct bmap* bmap_alloc(u32 max)
{
	struct bmap *bmap = fs_xzalloc(sizeof(*bmap) + max/8 + 8);
	bmap->len = max;
	return bmap;
}

void bmap_free(struct bmap *bmap)
{
	fs_xfree(bmap);
}

void bmap_insert(struct bmap *bmap, int x)
{
	u32 i = ((unsigned int)x) / 64;
	u32 j = ((unsigned int)x) % 64;
	u64 b = 1ULL << j;

	bmap->data[i] |= b;
}

void bmap_delete(struct bmap *bmap, int x)
{
	u32 i = ((unsigned int)x) / 64;
	u32 j = ((unsigned int)x) % 64;
	u64 b = 1ULL << j;

	bmap->data[i] &= ~b;
}

bool bmap_contains(struct bmap *bmap, int x)
{
	u32 i = ((unsigned int)x) / 64;
	u32 j = ((unsigned int)x) % 64;
	u64 b = 1ULL << j;

	return (bmap->data[i] & b) != 0;
}

struct bmap_iter
{
	struct bmap *bmap;
	u32 pos;
};

struct bmap_iter* bmap_iter_start(struct bmap *bmap)
{
	struct bmap_iter *i = fs_xmalloc(sizeof(*i));
	i->bmap = bmap;
	i->pos = ~0U;
	return i;
}

void bmap_iter_end(struct bmap_iter *i)
{
	fs_xfree(i);
}

bool bmap_iter_next(struct bmap_iter *iter, int *x_out)
{
	struct bmap *bmap = iter->bmap;
	do {
		u32 x = ++iter->pos;
		u32 i = x / 64;
		u32 j = x % 64;
		u64 b = 1ULL << j;

		if ((bmap->data[i] & b) != 0) {
			*x_out = (int)iter->pos;
			return true;
		}
	} while (iter->pos != bmap->len - 1);

	return false;
}
