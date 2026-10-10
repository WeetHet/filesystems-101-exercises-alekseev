package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"

	"tests-runner/util/errors"
	"tests-runner/util/util/ctxutil"
	"tests-runner/util/util/errutil"
	"tests-runner/util/workgroup"

	"tests-runner/db"
	"tests-runner/exercises"
	"tests-runner/git"
)

type VerificationQueue struct {
	s *Server

	ctx       context.Context
	cancelCtx ctxutil.CancelReasonFunc
	wg        *workgroup.Wg
}

func (s *Server) newVerificationQueue() *VerificationQueue {
	return &VerificationQueue{s: s}
}

func (vq *VerificationQueue) Start(ctx context.Context) error {
	vq.ctx, vq.cancelCtx = ctxutil.WithCancelReason(ctx)
	vq.wg = workgroup.New(workgroup.Config{
		Concurrency: 1,
		StopOnError: false,
	})
	return nil
}

func (vq *VerificationQueue) Stop() {
	vq.cancelCtx(errors.Errorf("server stopped"))
}

func (vq *VerificationQueue) Wait() {
	vq.wg.Wait()
}

func (vq *VerificationQueue) Submit(repo *db.Repo, s *db.Submission) {
	vq.wg.GoCtx(vq.ctx, func(ctx context.Context) error {
		vq.handleSubmission(ctx, repo, s)
		return nil
	})
}

func (vq *VerificationQueue) handleSubmission(ctx context.Context, repo *db.Repo, s *db.Submission) {
	log := vq.s.log.WithField("s", s.Id)

	var err error
	defer func() {
		if err != nil {
			log.WithError(err).Errorf(ctx, "verification failed")
			vq.reportErr(ctx, s, err)
		} else {
			log.Infof(ctx, "verification succeeded")
		}
	}()

	// 0. Do some minor preparations first.

	test, ok := exercises.Exercises[s.ExerciseId]
	if !ok {
		err = vq.report(ctx, s, &exercises.SubmissionResult{
			Id:     s.Id,
			Result: fmt.Sprintf("no such exercise: %q", s.ExerciseId),
		})
		return
	}

	ghUserName, err := vq.s.repoUrl2UserName(repo.Url)
	repoDir := path.Join(vq.s.reposDir, ghUserName)

	vq.s.lockRepo(repo.Url)
	defer vq.s.unlockRepo(repo.Url)

	// 1. Checkout a commit to be verified.

	if err = git.CheckoutTempBranch(repoDir, s.Id.String(), s.CommitId); err != nil {
		return
	}

	// 2. Update the solution to write output to a file.

	solutionDir := path.Join(repoDir, s.ExerciseId)
	if d, err := os.Open(solutionDir); err != nil {
		err = vq.report(ctx, s, &exercises.SubmissionResult{
			Id:     s.Id,
			Result: fmt.Sprintf("directory not found in the solution repo: %q", s.ExerciseId),
		})
		return
	} else {
		d.Close()
	}

	err = errutil.First(
		copyDir(test.AuxSrc, solutionDir),
		copyDir(exercises.AuxSrcStdlib, path.Join(repoDir, "stdlib")),
	)
	if err != nil {
		err = errors.Wrap(err, "replaceCallbacks()")
		return
	}

	// 3. Compile the solution.

	exerciseDir := path.Join(repoDir, s.ExerciseId)

	makeCmd := exec.Command("make", "build")
	makeCmd.Dir = exerciseDir

	compilerOutput, err := makeCmd.Output()
	if exit := new(exec.ExitError); errors.As(err, &exit) && exit.ExitCode() != 0 {
		log.Infof(ctx, "compilation failed")

		compilerOutput = append(compilerOutput, exit.Stderr...)
		err = vq.report(ctx, s, &exercises.SubmissionResult{
			Id:             s.Id,
			Result:         "compilation failed",
			CompilerOutput: string(compilerOutput),
		})
		return
	}
	if err != nil {
		return
	}

	// 4. Finally, run tests.

	result, err := test.Verify(ctx, s, exerciseDir)
	if err != nil {
		return
	}

	err = vq.report(ctx, s, result)
	return
}

func (vq *VerificationQueue) reportErr(ctx context.Context, s *db.Submission, err error) error {
	return vq.report(ctx, s, &exercises.SubmissionResult{
		Id:     s.Id,
		Result: fmt.Sprintf("internal error: %s", err.Error()),
	})
}

func (vq *VerificationQueue) report(ctx context.Context, s *db.Submission, r *exercises.SubmissionResult) error {
	resultText, err := json.Marshal(r)
	errutil.FatalIf(err)

	return vq.s.db.SetSubmissionResult(s, string(resultText), r.Result, r.Result == "ok")
}

func copyDir(src fs.FS, dst string) (err error) {
	files, err := fs.ReadDir(src, ".")
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.Name() == "." || f.Name() == ".." {
			continue
		}

		p := path.Join(dst, f.Name())

		t := f.Type() & os.ModeType
		if t == os.ModeDir {
			err = os.MkdirAll(p, 0700)
			if err != nil {
				return err
			}

			subsrc, err := fs.Sub(src, f.Name())
			if err != nil {
				return err
			}

			err = copyDir(subsrc, p)
			if err != nil {
				return err
			}

			continue
		}

		if t != 0 {
			return errors.ErrInvalidArg.WithMessagef("not a regular file or directory: %q", f.Name())
		}

		b, err := fs.ReadFile(src, f.Name())
		if err != nil {
			return err
		}

		err = os.WriteFile(p, b, 0777)
		if err != nil {
			return err
		}
	}

	return nil
}
