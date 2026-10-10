package exercise

import (
	"embed"
	"io/fs"

	"tests-runner/exercises"

	"tests-runner/util/util/errutil"
)

var (
	//go:embed src/*
	__auxSrc embed.FS

	//go:embed "testdata/00-trivial"
	t00dump  []byte
	t00dumpF = exercises.ExpectedOutputFile{
		Name:      "dump",
		IfMissing: "solution never created an output file",
		Verifier:  exercises.BytesEqual(t00dump),
	}
	t00 = exercises.Testcase{
		Name:            "00-trivial",
		Args:            []string{"test", "00-trivial"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t00dumpF},
	}

	//go:embed "testdata/01-hello"
	t01dump  []byte
	t01dumpF = exercises.ExpectedOutputFile{
		Name:      "dump",
		IfMissing: "solution never created an output file",
		Verifier:  exercises.BytesEqual(t01dump),
	}
	t01 = exercises.Testcase{
		Name:            "01-hello",
		Args:            []string{"test", "01-hello"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t01dumpF},
	}

	//go:embed "testdata/02-enoent"
	t02err  []byte
	t02errF = exercises.ExpectedOutputFile{
		Name:      "err",
		IfMissing: "solution never reported a problem",
		Verifier:  exercises.BytesEqual(t02err),
	}
	t02 = exercises.Testcase{
		Name:            "02-enoent",
		Args:            []string{"test", "02-enoent"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t02errF},
	}

	//go:embed "testdata/03-enotdir"
	t03err  []byte
	t03errF = exercises.ExpectedOutputFile{
		Name:      "err",
		IfMissing: "solution never reported a problem",
		Verifier:  exercises.BytesEqual(t03err),
	}
	t03 = exercises.Testcase{
		Name:            "03-enotdir",
		Args:            []string{"test", "03-enotdir"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t03errF},
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "07-ntfs-read-file",
		AuxSrc:    auxSrc,
		Testcases: []exercises.Testcase{t00, t01, t02, t03},
	})
}
