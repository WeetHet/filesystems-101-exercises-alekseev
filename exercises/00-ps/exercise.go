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

	//go:embed "testdata/t00-exe"
	t00exe string
	//go:embed "testdata/t00-argv"
	t00argv string
	//go:embed "testdata/t00-envp"
	t00envp string

	t00exeF = exercises.ExpectedOutputFile{
		Name:      "exe",
		IfMissing: "solution never called report_process()",
		Verifier:  exercises.LinesEqual(t00exe).Verify,
	}
	t00argvF = exercises.ExpectedOutputFile{
		Name:      "argv",
		IfMissing: "solution never called report_process()",
		Verifier:  exercises.LinesEqualS(t00argv).Verify,
	}
	t00envpF = exercises.ExpectedOutputFile{
		Name:      "envp",
		IfMissing: "solution never called report_process()",
		Verifier:  exercises.LinesEqualS(t00envp).Verify,
	}
	t00 = exercises.Testcase{
		Name:            "00-trivial",
		Args:            []string{"test", "00-trivial"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t00exeF, t00argvF, t00envpF},
	}

	//go:embed "testdata/t01-exe"
	t01exe string
	//go:embed "testdata/t01-argv"
	t01argv string
	//go:embed "testdata/t01-envp"
	t01envp string

	t01exeF = exercises.ExpectedOutputFile{
		Name:      "exe",
		IfMissing: "solution never called report_process()",
		Verifier:  exercises.LinesEqual(t01exe).Verify,
	}
	t01argvF = exercises.ExpectedOutputFile{
		Name:      "argv",
		IfMissing: "solution never called report_process()",
		Verifier:  exercises.LinesEqualS(t01argv).Verify,
	}
	t01envpF = exercises.ExpectedOutputFile{
		Name:      "envp",
		IfMissing: "solution never called report_process()",
		Verifier:  exercises.LinesEqualS(t01envp).Verify,
	}
	t01 = exercises.Testcase{
		Name:            "01-long-argv",
		Args:            []string{"test", "01-long-argv"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t01exeF, t01argvF, t01envpF},
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

	//go:embed "testdata/t03-exe"
	t03exe string
	//go:embed "testdata/t03-argv"
	t03argv string
	//go:embed "testdata/t03-envp"
	t03envp string

	t03exeF = exercises.ExpectedOutputFile{
		Name:      "exe",
		IfMissing: "solution never called report_process()",
		Verifier:  exercises.LinesEqual(t03exe).Verify,
	}
	t03argvF = exercises.ExpectedOutputFile{
		Name:      "argv",
		IfMissing: "solution never called report_process()",
		Verifier:  exercises.LinesEqualS(t03argv).Verify,
	}
	t03envpF = exercises.ExpectedOutputFile{
		Name:      "envp",
		IfMissing: "solution never called report_process()",
		Verifier:  exercises.LinesEqualS(t03envp).Verify,
	}
	t03 = exercises.Testcase{
		Name:            "03-long-arg",
		Args:            []string{"test", "03-long-arg"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t03exeF, t03argvF, t03envpF},
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "00-ps",
		AuxSrc:    auxSrc,
		Testcases: []exercises.Testcase{t00, t01, t02, t03},
	})
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
