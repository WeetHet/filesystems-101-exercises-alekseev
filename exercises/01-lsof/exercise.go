package exercise

import (
	"bufio"
	"embed"
	"io"
	"io/fs"
	"log"
	"regexp"
	"strings"

	"tests-runner/exercises"

	"tests-runner/util/container/set"
	"tests-runner/util/util/errutil"
)

var (
	//go:embed src/*
	__auxSrc embed.FS

	//go:embed "testdata/t00-files"
	t00files string

	t00filesF = exercises.ExpectedOutputFile{
		Name:      "files",
		IfMissing: "solution never called report_file()",
		Verifier:  verifyFiles(t00files),
	}
	t00 = exercises.Testcase{
		Name:            "00-trivial",
		Args:            []string{"test", "00-trivial"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t00filesF},
	}

	//go:embed "testdata/t01-files"
	t01files string

	t01filesF = exercises.ExpectedOutputFile{
		Name:      "files",
		IfMissing: "solution never called report_file()",
		Verifier:  verifyFiles(t01files),
	}
	t01 = exercises.Testcase{
		Name:            "01-long-path",
		Args:            []string{"test", "01-long-path"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t01filesF},
	}

	t02errF = exercises.ExpectedOutputFile{
		Name:      "err",
		IfMissing: "solution never called report_error()",
		Verifier:  t02verifyErr,
	}
	t02 = exercises.Testcase{
		Name:            "02-enoent",
		Args:            []string{"test", "02-enoent"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t02errF},
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "01-lsof",
		AuxSrc:    auxSrc,
		Testcases: []exercises.Testcase{t00, t01, t02},
	})
}

func verifyFiles(mustFilesStr string) exercises.VerifierFunc {
	mustFiles := strings.Split(mustFilesStr, "\n")

	return func(file io.Reader) (r *exercises.TestResult, err error) {
		files := set.NewStrSet()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			files.Add(scanner.Text())
		}
		if err = scanner.Err(); err != nil {
			return nil, err
		}

		for _, x := range mustFiles {
			if x != "" && !files.Contains(x) {
				return exercises.WrongAnswer("too few files reported"), nil
			}
		}

		return exercises.CorrectAnswer(), nil
	}
}

func t02verifyErr(file io.Reader) (r *exercises.TestResult, err error) {
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	expected := regexp.MustCompile("/proc/.*\nNo such file or directory")
	if expected.Match(b) {
		return exercises.CorrectAnswer(), nil
	} else if strings.Contains(string(b), "No such file or directory") {
		return exercises.WrongAnswer("ENOENT reported, but file path is wrong"), nil
	} else {
		return exercises.WrongAnswer("no report about ENOENT"), nil
	}
}

func t03verifyErr(file io.Reader) (r *exercises.TestResult, err error) {
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	log.Printf("!!! %q", string(b))

	expected := regexp.MustCompile("/proc/42.*\nNo such file or directory")
	if expected.Match(b) {
		return exercises.CorrectAnswer(), nil
	} else if strings.Contains(string(b), "No such file or directory") {
		return exercises.WrongAnswer("ENOENT reported, but file path is wrong"), nil
	} else {
		return exercises.WrongAnswer("no report about ENOENT"), nil
	}
}
