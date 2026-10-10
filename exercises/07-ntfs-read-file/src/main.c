#include <solution.h>
#include <fs_stdio.h>

#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <err.h>

int main(int argc, char **argv)
{
	if (argc != 3 || strcmp(argv[1], "test") != 0)
		errx(1, "invalid invocation");

	int img = open("img", O_RDONLY);
	int out = open("/out/dump", O_WRONLY|O_CREAT|O_TRUNC, S_IRUSR|S_IRGRP|S_IROTH);

	if (img < 0)
		errx(1, "open(img) failed");
	if (out < 0)
		errx(1, "open(out) failed");

	const char *path;
	int expect_error;
	if (strcmp(argv[2], "00-trivial") == 0)
		path = "/00-trivial", expect_error = 0;
	else if (strcmp(argv[2], "01-hello") == 0)
		path = "/d/01-hello", expect_error = 0;
	else if (strcmp(argv[2], "02-enoent") == 0)
		path = "/d/no-such-dir/x", expect_error = 1;
	else if (strcmp(argv[2], "03-enotdir") == 0)
		path = "/00-trivial/hello", expect_error = 1;
	else
		errx(1, "invalid invocation");

	int r = dump_file(img, path, out);
	if (r > 0)
		errx(1, "dump_file() must return negative error codes");
	if (r < 0 && !expect_error)
		errx(1, "dump_file() returned a non-zero result: %i", r);
	if (r < 0 && expect_error) {
		FILE *f = fs_fopen("/out/err", "w");
		fs_fprintf(f, "%s\n", strerror(-r));
		fs_fclose(f);
	}

	close(out);
	close(img);

	return 0;
}
