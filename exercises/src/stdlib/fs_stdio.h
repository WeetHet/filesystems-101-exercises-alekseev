#pragma once

#include <stdio.h>
#include <sys/types.h>

FILE* fs_fopen(const char *path, const char *mode);
void fs_fprintf(FILE *f, const char *fmt, ...);
void fs_fwrite(FILE *f, const void *buf, size_t len);
void fs_fclose(FILE *f);
