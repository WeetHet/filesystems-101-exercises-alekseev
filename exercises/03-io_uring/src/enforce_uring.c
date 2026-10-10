#include <enforce_uring.h>

#include <valgrind.h>
#include <liburing.h>
#include <stdio.h>
#include <stdlib.h>


bool io_uring_submit_called;


static void validate_io_uring_opcodes(struct io_uring *ring)
{
	struct io_uring_sq *sq = &ring->sq;
	for (unsigned i = sq->sqe_head; i < sq->sqe_tail; ++i) {
		struct io_uring_sqe *sqe = &sq->sqes[i];
		if (sqe->opcode == IORING_OP_READV || sqe->opcode == IORING_OP_WRITEV) {
			fprintf(stderr, "don't use readv() and writev()");
			exit(1);
		}
	}
}

int I_WRAP_SONAME_FNNAME_ZZ(liburingZdsoZa,ioZuuringZusubmitZa)(struct io_uring *ring)
{
	io_uring_submit_called = true;

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

	int res;
	OrigFn fn;
	VALGRIND_GET_ORIG_FN(fn);
	validate_io_uring_opcodes(ring);
	CALL_FN_W_WW(res, fn, ring, wait_nr);
	return res;
}
