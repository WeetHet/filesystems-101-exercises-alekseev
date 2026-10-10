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
	if (img < 0)
		errx(1, "open(img) failed");

	int inode_nr;
	if (strcmp(argv[2], "002") == 0)
		inode_nr = 2;
	else
		errx(1, "invalid invocation");

	int r = dump_dir(img, inode_nr);
	if (r < 0)
		errx(1, "dump_dir() failed");

	close(img);
	return 0;
}
