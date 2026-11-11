#include <enforce_uring.h>

#include <valgrind.h>
#include <liburing.h>
#include <err.h>


bool io_uring_submit_called;

int fail_submit_at, fail_io_at;
static int nr_submits;


static void validate_io_uring_opcodes(struct io_uring *ring)
{
	struct io_uring_sq *sq = &ring->sq;
	for (unsigned i = sq->sqe_head; i < sq->sqe_tail; ++i) {
		struct io_uring_sqe *sqe = &sq->sqes[i];
		if (sqe->opcode == IORING_OP_READV || sqe->opcode == IORING_OP_WRITEV)
			errx(1, "don't use IORING_OP_READV and IORING_OP_WRITEV");
	}
}

static void corrupt_sqe(struct io_uring *ring)
{
	struct io_uring_sq *sq = &ring->sq;
	unsigned head = sq->sqe_head, tail = sq->sqe_tail;
	struct io_uring_sqe *sqe;

	if (tail == head)
		return;

	sqe = &sq->sqes[(tail-1) & sq->ring_mask];
	sqe->addr = -1;
}

int I_WRAP_SONAME_FNNAME_ZZ(liburingZdsoZa,ioZuuringZusubmitZa)(struct io_uring *ring)
{
	io_uring_submit_called = true;

	int nr = ++nr_submits;
	if (nr == fail_submit_at)
		return -ENOMEM;
	if (nr == fail_io_at)
		corrupt_sqe(ring);

	int res;
	OrigFn fn;
	VALGRIND_GET_ORIG_FN(fn);
	validate_io_uring_opcodes(ring);
	CALL_FN_W_W(res, fn, ring);
	return res;
}

int I_WRAP_SONAME_FNNAME_ZZ(liburingZdsoZa,ioZuuringZusubmitZuandZuwaitZa)(struct io_uring *ring, int wait_nr)
{
	io_uring_submit_called = true;

	int nr = ++nr_submits;
	if (nr == fail_submit_at)
		return -ENOMEM;
	if (nr == fail_io_at)
		corrupt_sqe(ring);

	int res;
	OrigFn fn;
	VALGRIND_GET_ORIG_FN(fn);
	validate_io_uring_opcodes(ring);
	CALL_FN_W_WW(res, fn, ring, wait_nr);
	return res;
}

int I_WRAP_SONAME_FNNAME_ZZ(liburingZdsoZa,ioZuuringZusubmitZuandZuwaitZutimeout)(void)
{
	errx(1, "just use io_uring_submit() and io_uring_submit_and_wait()");
}

int I_WRAP_SONAME_FNNAME_ZZ(liburingZdsoZa,ioZuuringZusubmitZuandZuwaitZuminZutimeout)(void)
{
	errx(1, "just use io_uring_submit() and io_uring_submit_and_wait()");
}

int I_WRAP_SONAME_FNNAME_ZZ(liburingZdsoZa,ioZuuringZusubmitZuandZuwaitZureg)(void)
{
	errx(1, "just use io_uring_submit() and io_uring_submit_and_wait()");
}
