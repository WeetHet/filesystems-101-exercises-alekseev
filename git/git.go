package git

import (
	"os"
	"os/exec"
	"path"
	"strings"

	"tests-runner/util/errors"
	"tests-runner/util/util/errutil"
)

func Clone(dir, url, dstDir string) (err error) {
	defer func() { err = errors.Wrapf(err, "git.Clone(%q)", url) }()

	return errutil.First(
		os.RemoveAll(path.Join(dir, dstDir)),
		gitCommand(dir, "clone", "-q", url, dstDir).Run(),
	)
}

func Pull(repoDir string) (err error) {
	defer func() { err = errors.Wrap(err, "git.Pull()") }()

	return gitCommand(repoDir, "pull", "-q").Run()
}

func CheckoutBranch(repoDir, branch string) (err error) {
	defer func() { err = errors.Wrapf(err, "git.CheckoutBranch(%q)", branch) }()

	return gitCommand(repoDir, "checkout", "-q", "-f", branch).Run()
}

func CheckoutTempBranch(repoDir, branch, commitId string) (err error) {
	defer func() { err = errors.Wrapf(err, "git.CheckoutTempBranch(%q)", commitId) }()

	return gitCommand(repoDir, "checkout", "-q", "-f", "-B", branch, commitId).Run()
}

func ResetHard(repoDir, ref string) (err error) {
	defer func() { err = errors.Wrapf(err, "git.ResetHard(%q)", ref) }()

	return gitCommand(repoDir, "reset", "--hard", "-q", ref).Run()
}

func GetRefHash(repoDir string, ref string) (hash string, err error) {
	defer func() { err = errors.Wrapf(err, "git.GetRefHash(%q)", ref) }()

	h, err := gitCommand(repoDir, "rev-parse", ref).Output()
	if err != nil {
		return "", err
	}

	hash = strings.TrimSuffix(string(h), "\n")
	return hash, nil
}

func Clean(repoDir string) (err error) {
	defer func() { err = errors.Wrap(err, "git.Clean()") }()

	return gitCommand(repoDir, "clean", "-q", "-f").Run()
}

func gitCommand(workDir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	return cmd
}
