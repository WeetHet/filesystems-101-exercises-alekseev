package exercise

import (
	"embed"
	"io"
	"io/fs"
	"regexp"
	"strings"

	"tests-runner/exercises"

	"tests-runner/util/util/errutil"
)

var (
	//go:embed src/*
	__auxSrc embed.FS

	t00F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/a.out").Verify,
	}
	t00 = exercises.Testcase{
		Name:            "00-trivial-abs",
		Args:            []string{"00-trivial-abs"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t00F},
	}

	t01F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/a.out").Verify,
	}
	t01 = exercises.Testcase{
		Name:            "01-trivial-rel",
		Args:            []string{"01-trivial-rel"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t01F},
	}

	t02F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/dir/").Verify,
	}
	t02 = exercises.Testcase{
		Name:            "02-trivial-dir",
		Args:            []string{"02-trivial-dir"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t02F},
	}

	t03F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/a.out").Verify,
	}
	t03 = exercises.Testcase{
		Name:            "03-dot",
		Args:            []string{"03-dot"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t03F},
	}

	t04F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/a.out").Verify,
	}
	t04 = exercises.Testcase{
		Name:            "04-dotdot",
		Args:            []string{"04-dotdot"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t04F},
	}

	t05F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/.dot").Verify,
	}
	t05 = exercises.Testcase{
		Name:            "05-dotfile",
		Args:            []string{"05-dotfile"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t05F},
	}

	t06F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/a.out").Verify,
	}
	t06 = exercises.Testcase{
		Name:            "06-/../a.out",
		Args:            []string{"06-/../a.out"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t06F},
	}

	t07F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/file").Verify,
	}
	t07 = exercises.Testcase{
		Name:            "07-symlink-rel-0",
		Args:            []string{"07-symlink-rel-0"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t07F},
	}

	t08F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/dir/").Verify,
	}
	t08 = exercises.Testcase{
		Name:            "08-symlink-rel-0",
		Args:            []string{"08-symlink-rel-1"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t08F},
	}

	t09F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_path()",
		Verifier:  exercises.LinesEqual("/file").Verify,
	}
	t09 = exercises.Testcase{
		Name:            "09-symlink-abs",
		Args:            []string{"09-symlink-abs"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t09F},
	}

	t10F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_error()",
		Verifier:  exercises.LinesEqual("/file").Verify,
	}
	t10 = exercises.Testcase{
		Name:            "10-symlink-in-symlink",
		Args:            []string{"10-symlink-in-symlink"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t10F},
	}

	t11F = exercises.ExpectedOutputFile{
		Name:      "path",
		IfMissing: "solution never called report_error()",
		Verifier:  exercises.LinesEqual("/dir/").Verify,
	}
	t11 = exercises.Testcase{
		Name:            "11-trailing-slash-0",
		Args:            []string{"11-trailing-slash-0"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t11F},
	}

	t12F = exercises.ExpectedOutputFile{
		Name:      "err",
		IfMissing: "solution never called report_error()",
		Verifier:  t12verifyErr,
	}
	t12 = exercises.Testcase{
		Name:            "12-enoent",
		Args:            []string{"12-enoent"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t12F},
	}

	t13F = exercises.ExpectedOutputFile{
		Name:      "err",
		IfMissing: "solution never called report_error()",
		Verifier:  t13verifyErr,
	}
	t13 = exercises.Testcase{
		Name:            "13-enotdir",
		Args:            []string{"13-enotdir"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t13F},
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	exercises.RegisterExercise(exercises.Exercise{
		Name:   "13-realpath",
		AuxSrc: auxSrc,
		Testcases: []exercises.Testcase{t00, t01, t02, t03, t04, t05,
			t06, t07, t08, t09, t10, t11, t12, t13},
	})
}

func t12verifyErr(file io.Reader) (r *exercises.TestResult, err error) {
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	expected := regexp.MustCompile("/\nabc\nNo such file or directory")
	if expected.Match(b) {
		return exercises.CorrectAnswer(), nil
	} else if strings.Contains(string(b), "No such file or directory") {
		return exercises.WrongAnswer("ENOENT reported, but the file path is wrong"), nil
	} else {
		return exercises.WrongAnswer("no report about ENOENT"), nil
	}
}

func t13verifyErr(file io.Reader) (r *exercises.TestResult, err error) {
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	expected0 := regexp.MustCompile("/\na.out\nNot a directory")
	expected1 := regexp.MustCompile("/a.out\nabc\nNot a directory")
	if expected0.Match(b) {
		return exercises.CorrectAnswer(), nil
	} else if expected1.Match(b) {
		return exercises.CorrectAnswer(), nil
	} else if strings.Contains(string(b), "Not a directory") {
		return exercises.WrongAnswer("ENOTDIR reported, but the file path is wrong"), nil
	} else {
		return exercises.WrongAnswer("no report about ENOTDIR"), nil
	}
}
