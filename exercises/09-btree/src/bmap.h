#pragma once

#include <types.h>

struct bmap;

struct bmap* bmap_alloc(u32 max);
void bmap_free(struct bmap *bmap);

void bmap_insert(struct bmap *bmap, int x);
void bmap_delete(struct bmap *bmap, int x);
bool bmap_contains(struct bmap *bmap, int x);

struct bmap_iter;

struct bmap_iter* bmap_iter_start(struct bmap *bmap);
void bmap_iter_end(struct bmap_iter *i);

bool bmap_iter_next(struct bmap_iter *i, int *x);
