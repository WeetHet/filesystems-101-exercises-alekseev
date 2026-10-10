#include <solution.h>

#include <fs_stdio.h>
#include <string.h>

void report_path(const char *path)
{
	FILE *f_path = fs_fopen("./out/path", "w");
	fs_fprintf(f_path, "%s", path);
	fs_fclose(f_path);
}

void report_error(const char *parent, const char *child, int errno_code)
{
	FILE *f_err = fs_fopen("./out/err", "w");
	fs_fprintf(f_err, "%s\n%s\n%s\n", parent, child, strerror(errno_code));
	fs_fclose(f_err);
}
