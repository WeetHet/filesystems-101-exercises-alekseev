#pragma once

#include <stdbool.h>

extern bool io_uring_submit_called;

extern int fail_submit_at, fail_io_at;
