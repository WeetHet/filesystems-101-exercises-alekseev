#include <solution.h>
#include <types.h>
#include <fs_malloc.h>
#include <bmap.h>

#include <string.h>
#include <stdlib.h>
#include <stdint.h>
#include <err.h>

#define MAX_KEY (64 * 1024 * 1024)
#define NR_ITERS (1024 * 1024)

static u64 xrandseed;
static u32 xrand(u32 range);

int main(int argc, char *argv[])
{
	if (argc != 3 || strcmp(argv[1], "test") != 0)
		errx(1, "invalid invocation");

	xrandseed = atoi(argv[2]);

	struct btree *t = btree_alloc(3 + xrand(100));
	struct bmap *bmap = bmap_alloc(MAX_KEY);

	u32 *values = fs_xmalloc(NR_ITERS * sizeof(u32));
	u32 nr_values = 0;

	const char *errmsg = NULL;

	for (int n = 0; n < NR_ITERS; ++n) {
		u32 x = xrand(MAX_KEY);
		u32 p = xrand(nr_values);
		bool present = bmap_contains(bmap, (int)x);

		switch (xrand(6)) {
		case 0: // add random
			btree_insert(t, (int)x);
			bmap_insert(bmap, (int)x);

			if (!present)
				values[nr_values++] = x;

			break;

		case 1: // add previously inserted
			if (nr_values != 0) {
				x = values[p];
				btree_insert(t, (int)x);
				bmap_insert(bmap, (int)x);
			}
			break;

		case 2: // remove random
			btree_delete(t, (int)x);
			bmap_delete(bmap, (int)x);

			break;

		case 3: // remove previously inserted
			if (nr_values != 0) {
				x = values[p];
				btree_delete(t, (int)x);
				bmap_delete(bmap, (int)x);
			}
			break;

		case 4: // test random
			if (btree_contains(t, (int)x) != bmap_contains(bmap, (int)x)) {
				errmsg = "btree_contains() reports a wrong result";
				goto out;
			}
			break;

		case 5: // test previously inserted
			if (nr_values != 0) {
				x = values[p];
				if (btree_contains(t, (int)x) != bmap_contains(bmap, (int)x)) {
					errmsg = "btree_contains() reports a wrong result";
					goto out;
				}
			}
			break;
		}
	}

	struct btree_iter *i = btree_iter_start(t);
	struct bmap_iter *j = bmap_iter_start(bmap);
	int x, y;
	for (;;) {
		if (!btree_iter_next(i, &x))
			break;
		if (!bmap_iter_next(j, &y)) {
			errmsg = "btree has too many values";
			goto out_iter;
		}
		if (x != y) {
			errmsg = "btree has different values";
			goto out_iter;
		}
	}
	if (bmap_iter_next(j, &y)) {
		errmsg = "btree has too few values";
		goto out_iter;
	}

out_iter:
	bmap_iter_end(j);
	btree_iter_end(i);
out:
	fs_xfree(values);
	bmap_free(bmap);
	btree_free(t);

	if (errmsg != NULL)
		errx(1, "%s", errmsg);

	return 0;
}

u32 xrand(u32 range)
{
	if (range == 0)
		return 0;

	xrandseed *= 6364136223846793005;
	xrandseed += 1442695040888963407;

	u32 x = (u32)xrandseed;
	return x % range;
}
