#include <valgrind.h>
#include <err.h>


char* I_WRAP_SONAME_FNNAME_ZZ(libcZdsoZa, chdir)(char *path)
{
	(void) path;

	errx(1, "do not use chdir() because it changes the state of the caller process implicitly; use openat() and friends");
}
