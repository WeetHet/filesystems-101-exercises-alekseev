package exercise

import (
	"embed"
	"fmt"
	"io/fs"
	"strconv"

	"tests-runner/util/util/errutil"

	"tests-runner/exercises"
)

var (
	//go:embed src/*
	__auxSrc embed.FS
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	testcases := make([]exercises.Testcase, 0)
	for i := 0; i < 16; i++ {
		// NOTE: random seeds must be odd so that test sequences
		// start with "insert random" and "delete random".
		testcases = append(testcases, exercises.Testcase{
			Name: fmt.Sprintf("%02d", i),
			Args: []string{"test", strconv.Itoa(2*i + 1)},
		})
	}

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "09-btree",
		AuxSrc:    auxSrc,
		Testcases: testcases,
	})
}
