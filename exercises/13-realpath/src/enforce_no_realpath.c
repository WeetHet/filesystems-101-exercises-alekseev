#include <enforce_no_realpath.h>

#include <valgrind.h>

bool realpath_called;


char* I_WRAP_SONAME_FNNAME_ZZ(libcZdsoZa, realpath)(char *path_in, char *path_out)
{
	realpath_called = true;

	char *res;
	OrigFn fn;
	VALGRIND_GET_ORIG_FN(fn);
	CALL_FN_W_WW(res, fn, path_in, path_out);
	return res;
}
