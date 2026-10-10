#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#if !defined(FUSE_USE_VERSION)

char *strcpy(char *restrict dst, const char *restrict src)
{
	(void) dst;
	(void) src;

	fprintf(stderr, "don't use strcpy()");
	exit(1);
}

char *stpcpy(char *restrict dst, const char *restrict src)
{
	(void) dst;
	(void) src;

	fprintf(stderr, "don't use stpcpy()");
	exit(1);
}

char *strcat(char *restrict dst, const char *restrict src)
{
	(void) dst;
	(void) src;

	fprintf(stderr, "use snprintf() or asprintf() instead of strcat()");
	exit(1);
}

char *strncpy(char *restrict dst, const char *restrict src, size_t sz)
{
	(void) dst;
	(void) src;
	(void) sz;

	fprintf(stderr, "don't use strncpy(), it does not null-terminate the result");
	exit(1);
}

char *stpncpy(char *restrict dst, const char *restrict src, size_t sz)
{
	(void) dst;
	(void) src;
	(void) sz;

	fprintf(stderr, "don't use stpncpy(), it does not null-terminate the result");
	exit(1);
}

int sprintf(char *restrict str, const char *restrict format, ...)
{
	(void) str;
	(void) format;

	fprintf(stderr, "don't use sprintf(), it does not limit the length of the output");
	exit(1);
}

off_t lseek(int fd, off_t offset, int whence)
{
	(void) fd;
	(void) offset;
	(void) whence;

	fprintf(stderr, "don't use lseek(), use pread() and pwrite() instead");
	exit(1);
}

#endif /* !defined(FUSE_USE_VERSION) */
