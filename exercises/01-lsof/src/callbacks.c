#include <solution.h>

#include <fs_stdio.h>
#include <string.h>

void report_file(const char *path)
{
	FILE *f_files = fs_fopen("./out/files", "a");
	fs_fprintf(f_files, "%s\n", path);
	fs_fclose(f_files);
}

void report_error(const char *path, int errno)
{
	FILE *f_err = fs_fopen("./out/err", "w");
	fs_fprintf(f_err, "%s\n%s\n", path, strerror(errno));
	fs_fclose(f_err);
}
