#include <solution.h>

#include <fs_stdio.h>
#include <string.h>

extern pid_t child_pid;

void report_process(pid_t pid, const char *exe, char **argv, char **envp)
{
	if (pid != child_pid)
		return;

	FILE *f_exe = fs_fopen("./out/exe", "w");
	fs_fprintf(f_exe, "%s", exe);
	fs_fclose(f_exe);

	FILE *f_argv = fs_fopen("./out/argv", "w");
	for (char **x = argv; *x != NULL; ++x)
		fs_fprintf(f_argv, "%s\n", *x);
	fs_fclose(f_argv);

	FILE *f_envp = fs_fopen("./out/envp", "w");
	for (char **x = envp; *x != NULL; ++x)
		fs_fprintf(f_envp, "%s\n", *x);
	fs_fclose(f_envp);
}

void report_error(const char *path, int errno)
{
	FILE *f_err = fs_fopen("./out/err", "w");
	fs_fprintf(f_err, "%s\n%s\n", path, strerror(errno));
	fs_fclose(f_err);
}
