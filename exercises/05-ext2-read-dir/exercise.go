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

	//go:embed "testdata/002"
	t00dump  string
	t00dumpF = exercises.ExpectedOutputFile{
		Name:      "dump",
		IfMissing: "solution never created an output file",
		Verifier:  exercises.LinesEqualS(t00dump).Verify,
	}
	t00 = exercises.Testcase{
		Name:            "00",
		Args:            []string{"test", "002"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t00dumpF},
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "05-ext2-read-dir",
		AuxSrc:    auxSrc,
		Testcases: []exercises.Testcase{t00},
	})
}
