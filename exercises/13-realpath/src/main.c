#include <solution.h>
#include <enforce_no_realpath.h>

#include <fs_stdio.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/stat.h>
#include <string.h>
#include <err.h>

int main(int argc, char **argv)
{
	if (argc != 2)
		errx(1, "bad invocation");

	if (strcmp(argv[1], "00-trivial-abs") == 0)
		abspath("/a.out");
	else if (strcmp(argv[1], "01-trivial-rel") == 0)
		abspath("a.out");
	else if (strcmp(argv[1], "02-trivial-dir") == 0)
		abspath("dir");
	else if (strcmp(argv[1], "03-dot") == 0)
		abspath("./a.out");
	else if (strcmp(argv[1], "04-dotdot") == 0)
		abspath("/dir/../a.out");
	else if (strcmp(argv[1], "05-dotfile") == 0)
		abspath("./.dot");
	else if (strcmp(argv[1], "06-/../a.out") == 0)
		abspath("/../a.out");
	else if (strcmp(argv[1], "07-symlink-rel-0") == 0)
		abspath("symlink0");
	else if (strcmp(argv[1], "08-symlink-rel-1") == 0)
		abspath("symlink1");
	else if (strcmp(argv[1], "09-symlink-abs") == 0)
		abspath("symlink2");
	else if (strcmp(argv[1], "10-symlink-in-symlink") == 0)
		abspath("symlink3");
	else if (strcmp(argv[1], "11-trailing-slash-0") == 0)
		abspath("/dir/");
	else if (strcmp(argv[1], "12-enoent") == 0)
		abspath("/abc/a.out");
	else if (strcmp(argv[1], "13-enotdir") == 0)
		abspath("/a.out/abc");
	else
		errx(1, "bad invocation");

	if (realpath_called)
		errx(1, "you are supposed to implement realpath() yourself");

	return 0;
}
