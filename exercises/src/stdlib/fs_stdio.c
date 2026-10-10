#include <fs_stdio.h>

#include <stdarg.h>
#include <err.h>

FILE* fs_fopen(const char *path, const char *mode)
{
	FILE *f = fopen(path, mode);
	if (f == NULL)
		errx(1, "fopen() failed");
	return f;
}

void fs_fprintf(FILE *f, const char *fmt, ...)
{
	va_list ap;
	int n;

	va_start(ap, fmt);
	n = vfprintf(f, fmt, ap);
	va_end(ap);

	if (n < 0)
		errx(1, "fprintf() failed");
}

void fs_fwrite(FILE *f, const void *buf, size_t len)
{
	int n = fwrite(buf, 1, len, f);
	if (n < 0)
		errx(1, "fwrite() failed");
	if ((size_t)n != len)
		errx(1, "fwrite() wrote too few bytes");
}

void fs_fclose(FILE *f)
{
	int r = fclose(f);
	if (r < 0)
		errx(1, "fclose() failed");
}
