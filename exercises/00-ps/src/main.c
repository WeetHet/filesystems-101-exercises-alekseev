#include <solution.h>

#include <unistd.h>
#include <string.h>
#include <err.h>
#include <pthread.h>
#include <time.h>
#include <stdlib.h>
#include <stdio.h>
#include <sys/stat.h>
#include <fcntl.h>

static void test(const char *test_name);
static void child(void) __attribute__((noreturn));

static void invoke_child(const char *test_name) __attribute__((noreturn));

static const char *exe = "/a.out";
pid_t child_pid;

int main(int argc, char **argv)
{
	if (argc == 3 && strcmp(argv[1], "test") == 0) {
		test(argv[2]);
		return 0;
	} else if (argc >= 2 && strcmp(argv[1], "child") == 0) {
		child();
	} else {
		errx(1, "bad invocation");
	}
}

static void test(const char *test_name)
{
	child_pid = fork();
	if (child_pid < 0) {
		errx(1, "fork() failed");
	} else if (child_pid == 0) {
		invoke_child(test_name);
	} else {
		/* sleep 10 ms to give time to the child to call execve() */
		struct timespec s = {.tv_sec = 0, .tv_nsec = 10*1000*1000};
		nanosleep(&s, NULL);

		if (strcmp(test_name, "02-enoent") == 0) {
			if (system("mount -B /tmp /proc"))
				errx(1, "umount /proc failed");

			char path[1024];
			snprintf(path, sizeof(path), "/proc/%ld", (long int)child_pid);
			if (mkdir(path, 0777) != 0)
				errx(1, "mkdir(%s) failed", path);
		}

		ps();
	}
}

static void invoke_child(const char *test_name)
{
	if (strcmp(test_name, "00-trivial") == 0) {
		char *argv[] = {(char *)exe, "child", "somearg", NULL};
		char *envp[] = {"A=a", "B=b", NULL};

		execve(exe, argv, envp);
		errx(1, "execve() failed");
	} else if (strcmp(test_name, "01-long-argv") == 0) {
#define N (1024)
		char *argv[2 + N + 1];
		char *envp[N + 1];

		argv[0] = (char *)exe;
		argv[1] = "child";
		for (int i = 2; i <= N+1; ++i)
			argv[i] = "a long arg string";
		argv[N + 2] = NULL;

		for (int i = 0; i < N; ++i)
			envp[i] = "LONG_VAR_NAME=long_var_value";
		envp[N] = NULL;
#undef N
		execve(exe, argv, envp);
		errx(1, "execve() failed");
	} else if (strcmp(test_name, "02-enoent") == 0) {
		char *argv[] = {(char *)exe, "child", NULL};
		char *envp[] = {NULL};

		execve(exe, argv, envp);
		errx(1, "execve() failed");
	} else if (strcmp(test_name, "03-long-arg") == 0) {
		char longarg[16384 + 1];
		memset(&longarg[0], 'x', sizeof(longarg) - 1);
		longarg[sizeof(longarg) - 1] = '\0';

		char *argv[] = {(char *)exe, "child", longarg, NULL};
		char *envp[] = {NULL};

		execve(exe, argv, envp);
		errx(1, "execve() failed");
	}else {
		errx(1, "invalid invocation");
	}
}

static void child(void)
{
	pthread_mutex_t m = PTHREAD_MUTEX_INITIALIZER;
	pthread_mutex_lock(&m);
	pthread_mutex_lock(&m);
	errx(1, "failed to deadlock");
}
