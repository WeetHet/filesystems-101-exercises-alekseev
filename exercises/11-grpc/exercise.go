package exercise

import (
	"embed"
	"io/fs"

	"tests-runner/exercises"

	"tests-runner/util/util/errutil"
)

var (
	//go:embed "src/*"
	__auxSrc embed.FS

	t00 = exercises.Testcase{
		Name:            "00",
		Args:            []string{},
		WithoutValgrind: true,
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "11-grpc",
		AuxSrc:    auxSrc,
		Testcases: []exercises.Testcase{t00},
	})
}
