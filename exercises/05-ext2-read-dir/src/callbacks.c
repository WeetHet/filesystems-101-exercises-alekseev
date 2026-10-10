#include <solution.h>

#include <fs_stdio.h>

void report_file(int inode_nr, char type, const char *name)
{
	FILE *f_dump = fs_fopen("./out/dump", "a");
	fs_fprintf(f_dump, "%04i %c %s\n", inode_nr, type, name);
	fs_fclose(f_dump);
}
