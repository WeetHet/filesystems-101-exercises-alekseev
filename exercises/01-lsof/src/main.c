#include <solution.h>

#include <sys/stat.h>
#include <sys/fsuid.h>
#include <unistd.h>
#include <fcntl.h>
#include <errno.h>
#include <err.h>
#include <string.h>
#include <stdlib.h>

int main(int argc, char **argv)
{
	if (argc != 3)
		errx(1, "invalid invocation");

	if (strcmp(argv[2], "00-trivial") == 0) {
		int fd0 = open("/a.out", O_RDONLY);
		if (fd0 < 0)
			errx(1, "open() failed");
		int fd1 = open("/proc", O_RDONLY|O_DIRECTORY);
		if (fd1 < 0)
			errx(1, "open() failed");

		lsof();

		close(fd1);
		close(fd0);

	} else if (strcmp(argv[2], "01-long-path") == 0) {
		int fd = open("/tmp", O_RDONLY|O_DIRECTORY);
		if (fd < 0)
			errx(1, "open(/tmp) failed");

		for (int i = 0; i < 1020; ++i) {
			if (mkdirat(fd, "dir", 0777) < 0 && errno != EEXIST)
				errx(1, "mkdir() failed");

			int fd_next = openat(fd, "dir", O_RDONLY|O_DIRECTORY);
			if (fd_next < 0)
				errx(1, "open(dir) failed");
			close(fd);
			fd = fd_next;
		}

		lsof();

		close(fd);

	} else if (strcmp(argv[2], "02-enoent") == 0) {
		if (system("mount -B /tmp /proc"))
			errx(1, "umount /proc failed");

		if (mkdir("/proc/42", 0777) != 0)
			errx(1, "mkdir failed");

		lsof();

	} else {
		errx(1, "invalid invocation");
	}

	return 0;
}
