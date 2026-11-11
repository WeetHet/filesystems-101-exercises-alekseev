package exercise

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"

	"tests-runner/exercises"

	"tests-runner/util/util/errutil"
)

var (
	//go:embed src/*
	__auxSrc embed.FS

	testLengths = []int{
		1024 * 1024,
		4*1024*1024 - 4096,
		512 * 1024,
		256 * 1024,
		256*1024 + 4096,
		256*1024 - 1023,
		1,
		0,
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	var testcases []exercises.Testcase
	for i, len := range testLengths {
		testcases = append(testcases, exercises.Testcase{
			Name: fmt.Sprintf("%02d", i),
			Args: []string{"test", fmt.Sprintf("%d", len), "0", "0"},
			ExpectedOutputs: []exercises.ExpectedOutputFile{{
				Name:      "out",
				IfMissing: "solution never created the output file",
				Verifier:  makeVerifier(len),
			}},
			WithoutTrackFds: true,
		})
	}

	testcases = append(testcases, exercises.Testcase{
		Name:            "submit-fail-1",
		Args:            []string{"test", "8393727", "1", "0"},
		ExpectedOutputs: nil,
		WithoutTrackFds: true,
	})
	testcases = append(testcases, exercises.Testcase{
		Name:            "submit-fail-2",
		Args:            []string{"test", "8393727", "2", "0"},
		ExpectedOutputs: nil,
		WithoutTrackFds: true,
	})

	testcases = append(testcases, exercises.Testcase{
		Name:            "io-fail-1",
		Args:            []string{"test", "8393727", "0", "1"},
		ExpectedOutputs: nil,
		WithoutTrackFds: true,
	})
	testcases = append(testcases, exercises.Testcase{
		Name:            "io-fail-2",
		Args:            []string{"test", "8393727", "0", "2"},
		ExpectedOutputs: nil,
		WithoutTrackFds: true,
	})

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "03-io_uring",
		AuxSrc:    auxSrc,
		Testcases: testcases,
	})
}

func makeVerifier(expectedLen int) exercises.VerifierFunc {
	return func(file io.Reader) (r *exercises.TestResult, err error) {
		b, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		if len(b) != expectedLen {
			return exercises.WrongAnswer("file size differs"), nil
		}

		expected := generateTestFile(expectedLen)
		if bytes.Compare(b, expected) != 0 {
			return exercises.WrongAnswer("file content does not match"), nil
		}

		return exercises.CorrectAnswer(), nil
	}
}

func generateTestFile(len int) []byte {
	res := make([]byte, len)
	x := uint64(0)
	for i := 0; i < len; i++ {
		res[i] = byte(x)
		x *= 6364136223846793005
		x += 1442695040888963407
	}
	return res
}
