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

	t00 = exercises.Testcase{
		Name:            "00-ls",
		Args:            []string{"test", "00-ls"},
		WithoutValgrind: true,
	}
	t01 = exercises.Testcase{
		Name:            "01-read",
		Args:            []string{"test", "01-read"},
		WithoutValgrind: true,
	}
	t02 = exercises.Testcase{
		Name:            "02-read-pid",
		Args:            []string{"test", "02-read-pid"},
		WithoutValgrind: true,
	}
	t03 = exercises.Testcase{
		Name:            "03-erofs-0",
		Args:            []string{"test", "03-erofs-0"},
		WithoutValgrind: true,
	}
	t04 = exercises.Testcase{
		Name:            "03-erofs-1",
		Args:            []string{"test", "03-erofs-1"},
		WithoutValgrind: true,
	}
	t05 = exercises.Testcase{
		Name:            "04-enoent",
		Args:            []string{"test", "04-enoent"},
		WithoutValgrind: true,
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "02-fuse-helloworld",
		AuxSrc:    auxSrc,
		Testcases: []exercises.Testcase{t00, t01, t02, t03, t04, t05},
	})
}
