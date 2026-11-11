#include <solution.h>
#include <enforce_uring.h>
#include <fs_malloc.h>

#include <unistd.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <stdint.h>
#include <errno.h>
#include <err.h>

int main(int argc, char **argv)
{
	if (argc != 5 || strcmp(argv[1], "test") != 0)
		errx(1, "invalid invocation");

	int in = open("/out/in", O_RDWR|O_CREAT|O_TRUNC, S_IRUSR|S_IWUSR);
	int out = open("/out/out", O_WRONLY|O_CREAT|O_TRUNC, S_IRUSR|S_IRGRP|S_IROTH);

	if (in < 0) {
		close(out);
		errx(1, "open(in) failed");
	}
	if (out < 0) {
		close(in);
		errx(1, "open(out) failed");
	}

	size_t len;
	sscanf(argv[2], "%zd", &len);

	unsigned char *buf = fs_xmalloc(len);
        uint64_t x = 0;
	for (size_t i = 0; i < len; ++i) {
		buf[i] = (unsigned char)x;
		x *= 6364136223846793005;
		x += 1442695040888963407;
	}

	int n = write(in, buf, len);
	if (n < 0 || (size_t)n != len) {
		fs_xfree(buf);
		close(out);
		close(in);
		errx(1, "write() failed");
	}

	sscanf(argv[3], "%d", &fail_submit_at);
	sscanf(argv[4], "%d", &fail_io_at);

	int r = copy(in, out);

	fs_xfree(buf);
	close(out);
	close(in);

	if (!io_uring_submit_called && len != 0)
		errx(1, "io_uring_submit() or io_uring_submit_and_wait() was never called");

	if (fail_submit_at > 0) {
		if (r != -ENOMEM)
			errx(1, "io_uring_submit() or io_uring_submit_and_wait() has failed, but that was never reported");
		r = 0;
	}

	if (fail_io_at > 0) {
		if (r != -EFAULT)
			errx(1, "an IO has failed but that was never reported");
		r = 0;
	}

	return r;
}
