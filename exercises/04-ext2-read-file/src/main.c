#include <solution.h>

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

	int inode_nr;
	if (strcmp(argv[2], "002") == 0)
		inode_nr = 2;
	else if (strcmp(argv[2], "016") == 0)
		inode_nr = 16;
	else if (strcmp(argv[2], "506") == 0)
		inode_nr = 506;
	else
		errx(1, "invalid invocation");

	int r = dump_file(img, inode_nr, out);
	if (r < 0)
		errx(1, "dump_file() failed");

	close(out);
	close(img);

	return 0;
}
