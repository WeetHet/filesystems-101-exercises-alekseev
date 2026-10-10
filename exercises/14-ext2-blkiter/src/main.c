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

	int inode_nr;
	if (strcmp(argv[2], "002") == 0)
		inode_nr = 2;
	else if (strcmp(argv[2], "016") == 0)
		inode_nr = 16;
	else if (strcmp(argv[2], "506") == 0)
		inode_nr = 506;
	else if (strcmp(argv[2], "736") == 0)
		inode_nr = 736;
	else if (strcmp(argv[2], "3070") == 0)
		inode_nr = 3070;
	else if (strcmp(argv[2], "nops") == 0)
		inode_nr = -1;
	else
		errx(1, "invalid invocation");

	if (inode_nr == -1) {
		ext2_blkiter_free(NULL);
		ext2_fs_free(NULL);
		return 0;
	}

	int img = open("img", O_RDONLY);
	FILE *out = fs_fopen("/out/blocks", "w");

	if (img < 0)
		errx(1, "open(img) failed");
	if (out == NULL)
		errx(1, "open(out) failed");

	struct ext2_fs *fs = NULL;
	struct ext2_blkiter *i = NULL;
	int r;

	if ((r = ext2_fs_init(&fs, img)))
		errx(1, "ext2_fs_init() failed");
	if ((r = ext2_blkiter_init(&i, fs, inode_nr)))
		errx(1, "ext2_blkiter_init() failed");

	for (;;) {
		int blkno;
		r = ext2_blkiter_next(i, &blkno);
		if (r < 0)
			errx(1, "ext2_blkiter_next() failed");
		else if (r > 0)
			fs_fprintf(out, "%i\n", blkno);
		else
			break;
	}

	ext2_blkiter_free(i);
	ext2_fs_free(fs);
	fs_fclose(out);

	return 0;
}
