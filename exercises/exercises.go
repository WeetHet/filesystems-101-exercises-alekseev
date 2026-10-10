package exercises

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"tests-runner/db"
	"tests-runner/docker"

	"tests-runner/util/errors"
	"tests-runner/util/ulid"
	"tests-runner/util/workgroup"
)

type SubmissionResult struct {
	Id             ulid.ULID     `json:"id"`
	Result         string        `json:"result,omitempty"`
	CompilerOutput string        `json:"compiler_output,omitempty"`
	TestResults    []*TestResult `json:"tests,omitempty"`
}

type TestResult struct {
	TestName string `json:"test"`
	Result   string `json:"result"`
	Output   string `json:"output,omitempty"`
	Message  string `json:"message,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	VgOutput string `json:"vg,omitempty"`
}

var Exercises = make(map[string]Exercise)

func RegisterExercise(e Exercise) {
	Exercises[e.Name] = e
}

type Exercise struct {
	Name      string
	AuxSrc    fs.FS
	Testcases []Testcase
}

type Testcase struct {
	Name            string
	Args            []string
	InputDir        string
	ExpectedOutputs []ExpectedOutputFile

	WithoutValgrind bool
	WithoutTrackFds bool
}

type ExpectedOutputFile struct {
	Name      string
	IfMissing string
	Verifier  VerifierFunc
}

func (e *Exercise) Verify(ctx context.Context, s *db.Submission, solutionDir string) (r *SubmissionResult, err error) {
	defer func() { err = errors.Wrapf(err, "Verify(%s)", e.Name) }()

	tag := strings.ToLower(fmt.Sprintf("%s:0", s.Id))
	if err = docker.Build(ctx, solutionDir, "Dockerfile", tag); err != nil {
		return nil, err
	}
	defer docker.Remove(ctx, tag)

	testResults := make([]*TestResult, len(e.Testcases))

	wg := workgroup.New(workgroup.Config{Concurrency: 16})
	for i := range e.Testcases {
		t := &e.Testcases[i]
		tRes := &testResults[i]

		wg.GoCtx(ctx, func(ctx context.Context) error {
			r, err := t.run(ctx, tag)
			if err == nil {
				*tRes = r
			}
			return err
		})
	}
	wg.Wait()

	if err = wg.Error(); err != nil {
		return nil, err
	}

	result := "ok"
	for i, r := range testResults {
		if !r.IsOk() {
			result = r.Result
			testResults = testResults[:i+1]
			break
		}
	}

	r = &SubmissionResult{
		Id:          s.Id,
		Result:      result,
		TestResults: testResults,
	}
	return r, nil
}

func (t *Testcase) run(ctx context.Context, img string) (r *TestResult, err error) {
	defer func() {
		if r != nil {
			r.TestName = t.Name
		}
		err = errors.Wrapf(err, "run(%s)", t.Name)
	}()

	outputDir, err := os.MkdirTemp("", fmt.Sprintf("%s-", img))
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outputDir)

	rc := docker.RunConfig{}
	if t.WithoutValgrind {
		rc.Exe = "/a.out"
	} else {
		trackFds := "--track-fds=yes"
		if t.WithoutTrackFds {
			trackFds = "--track-fds=no"
		}

		rc.Exe = "/usr/bin/valgrind"
		rc.Args = append(rc.Args,
			"-q", "--log-file=/out/__vg",
			"--error-exitcode=2", "--exit-on-first-error=yes",
			"--tool=memcheck", "--leak-check=full", "--track-origins=yes",
			trackFds,
			"/a.out",
		)
	}
	rc.Args = append(rc.Args, t.Args...)

	rc.OutputDir = outputDir
	run, err := docker.Run(ctx, img, rc)
	if err != nil {
		return nil, err
	}

	if run.ExitCode == 2 && !t.WithoutValgrind {
		return VgDirty(readAll(outputDir, "__vg")), nil
	}
	if run.WasKilled() {
		return Killed("solution took too much time, or OOMed"), nil
	}
	if run.ExitCode != 0 {
		return NonZeroExitCode(run.ExitCode, run.Stderr), nil
	}

	for i := range t.ExpectedOutputs {
		f := &t.ExpectedOutputs[i]
		r, err := f.Verify(outputDir)
		if err != nil {
			return nil, err
		}
		r.Output = f.Name
		if !r.IsOk() {
			return r, nil
		}
	}

	return CorrectAnswer(), nil
}

func readAll(dir, file string) (res string) {
	f, err := os.Open(path.Join(dir, file))
	if err != nil {
		return err.Error()
	}
	defer f.Close()

	r, err := io.ReadAll(io.LimitReader(f, 256*1024))
	if err != nil {
		return err.Error()
	}
	return string(r)
}
