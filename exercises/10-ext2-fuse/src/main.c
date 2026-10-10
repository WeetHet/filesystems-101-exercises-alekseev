#include <solution.h>
#include <fs_stdio.h>

#include <stdio.h>
#include <string.h>
#include <unistd.h>
#include <stdlib.h>
#include <fcntl.h>
#include <sys/random.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include <dirent.h>
#include <time.h>
#include <errno.h>
#include <err.h>

static void run_test(const char *test_name);

static char mntp[128];

int main(int argc, char **argv)
{
	if (argc != 3 || strcmp(argv[1], "test") != 0)
		errx(1, "invalid invocation");

	long long int rnd = 0;
	getrandom(&rnd, sizeof(rnd), GRND_NONBLOCK);
	snprintf(mntp, sizeof(mntp), "/tmp/fs101ex-%016llx", rnd);

	if (mkdir(mntp, 0777) != 0)
		errx(1, "mkdir() failed");

	int img = open("/img", O_RDONLY);
	if (img < 0)
		errx(1, "open(img) failed");

	pid_t fuse_pid = fork();
	if (fuse_pid < 0) {
		errx(1, "fork() failed");
	} else if (fuse_pid == 0) {
		_exit(ext2fuse(img, mntp));
	} else {
		// give FUSE some time to mount
		struct timespec delay = {.tv_sec = 0, .tv_nsec = 20*1000*1000};
		nanosleep(&delay, NULL);

		run_test(argv[2]);

		return 0;
	}
}

static void run_test(const char *test_name)
{
	if (strcmp(test_name, "00-ls") == 0) {
		DIR *d = opendir(mntp);
		if (d == NULL)
			errx(2, "opendir() failed");

		FILE *ls = fs_fopen("/out/ls", "w");

		struct dirent *e;
		while ((e = readdir(d))) {
			char t;
			if (e->d_type == DT_REG)
				t = 'f';
			else if (e->d_type == DT_DIR)
				t = 'd';
			else
				t = '?';

			fs_fprintf(ls, "%c %s\n", t, e->d_name);
		}
		closedir(d);

		fs_fclose(ls);
	} else if (strcmp(test_name, "01-read-helloworld") == 0) {
		char path[2048];
		snprintf(path, sizeof(path), "%s/000/001/002/003/004/005/006/007/008/009/010/011/012/013/014/015/016/017/018/019/020/021/022/023/024/025/026/027/028/029/030/031/032/033/034/035/036/037/038/039/040/041/042/043/044/045/046/047/048/049/050/051/052/053/054/055/056/057/058/059/060/061/062/063/064/065/066/067/068/069/070/071/072/073/074/075/076/077/078/079/080/081/082/083/084/085/086/087/088/089/090/091/092/093/094/095/096/097/098/099/100/101/102/103/104/105/106/107/108/109/110/111/112/113/114/115/116/117/118/119/120/121/122/123/124/125/126/127/128/129/130/131/132/133/134/135/136/137/138/139/140/141/142/143/144/145/146/147/148/149/150/151/152/153/154/155/156/157/158/159/160/161/162/163/164/165/166/167/168/169/170/171/172/173/174/175/176/177/178/179/180/181/182/183/184/185/186/187/188/189/190/191/192/193/194/195/196/197/198/199/200/201/202/203/204/205/206/207/208/209/210/211/212/213/214/215/216/217/218/219/220/221/222/223/224/225/226/227/228/229/230/231/232/233/234/235/236/237/238/239/240/241/242/243/244/245/246/247/248/249/250/251/252/253/254/255/hello", mntp);

		int fd = open(path, O_RDONLY);
		if (fd < 0)
			errx(2, "open() failed");

		char buf[32];
		int n = pread(fd, buf, sizeof(buf), 0);
		if (n < 0)
			errx(2, "read() failed");

		close(fd);

		FILE *dump = fs_fopen("/out/dump-hello", "w");
		fs_fwrite(dump, buf, n);
		fs_fclose(dump);
	} else if (strcmp(test_name, "02-read-2ind") == 0) {
		char path[256];
		snprintf(path, sizeof(path), "%s/2ind", mntp);

		int fd = open(path, O_RDONLY);
		if (fd < 0)
			errx(2, "open() failed");

		char buf[4096];
		FILE *dump;
		int n;

		n = pread(fd, buf, sizeof(buf), 21*1024);
		if (n < 0)
			errx(2, "read() failed");

		dump = fs_fopen("/out/dump-singly-indirect", "w");
		fs_fwrite(dump, buf, n);
		fs_fclose(dump);

		n = pread(fd, buf, sizeof(buf), 3*1024);
		if (n < 0)
			errx(2, "read() failed");

		dump = fs_fopen("/out/dump-direct", "w");
		fs_fwrite(dump, buf, n);
		fs_fclose(dump);

		n = pread(fd, buf, sizeof(buf), 3*1024*1024 + 128);
		if (n < 0)
			errx(2, "read() failed");

		dump = fs_fopen("/out/dump-doubly-indirect", "w");
		fs_fwrite(dump, buf, n);
		fs_fclose(dump);

		close(fd);
	} else if (strcmp(test_name, "03-enoent") == 0) {
		char path[256];
		snprintf(path, sizeof(path), "%s/no_such_file", mntp);

		int fd = open(path, O_RDONLY);
		if (fd >= 0 || errno != ENOENT)
			errx(2, "enoent was not reported");
	} else if (strcmp(test_name, "04-enotdir") == 0) {
		char path[256];
		snprintf(path, sizeof(path), "%s/1023.f/file", mntp);

		int fd = open(path, O_RDONLY);
		if (fd >= 0 || errno != ENOTDIR)
			errx(2, "enotdir was not reported");
	} else {
		errx(1, "invalid invocation");
	}
}
