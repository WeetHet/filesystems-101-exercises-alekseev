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

	//go:embed "testdata/506-hello"
	t00blocks  string
	t00blocksF = exercises.ExpectedOutputFile{
		Name:      "blocks",
		IfMissing: "solution never created an output file",
		Verifier:  exercises.LinesEqualS(t00blocks).Verify,
	}
	t00 = exercises.Testcase{
		Name:            "00-helloworld",
		Args:            []string{"test", "506"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t00blocksF},
	}

	//go:embed "testdata/002-root"
	t01blocks  string
	t01blocksF = exercises.ExpectedOutputFile{
		Name:      "blocks",
		IfMissing: "solution never created an output file",
		Verifier:  exercises.LinesEqualS(t01blocks).Verify,
	}
	t01 = exercises.Testcase{
		Name:            "01-single-indirect",
		Args:            []string{"test", "002"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t01blocksF},
	}

	//go:embed "testdata/016-2ind"
	t02blocks  string
	t02blocksF = exercises.ExpectedOutputFile{
		Name:      "blocks",
		IfMissing: "solution never created an output file",
		Verifier:  exercises.LinesEqualS(t02blocks).Verify,
	}
	t02 = exercises.Testcase{
		Name:            "02-double-indirect",
		Args:            []string{"test", "016"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t02blocksF},
	}

	//go:embed "testdata/736-empty"
	t03blocks  string
	t03blocksF = exercises.ExpectedOutputFile{
		Name:      "blocks",
		IfMissing: "solution never created an output file",
		Verifier:  exercises.LinesEqualS(t03blocks).Verify,
	}
	t03 = exercises.Testcase{
		Name:            "03-empty",
		Args:            []string{"test", "736"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t03blocksF},
	}

	t04 = exercises.Testcase{
		Name: "04-nops",
		Args: []string{"test", "nops"},
	}

	//go:embed "testdata/3070-bg1"
	t05blocks  string
	t05blocksF = exercises.ExpectedOutputFile{
		Name:      "blocks",
		IfMissing: "solution never created an output file",
		Verifier:  exercises.LinesEqualS(t05blocks).Verify,
	}
	t05 = exercises.Testcase{
		Name:            "05-bg#1",
		Args:            []string{"test", "3070"},
		ExpectedOutputs: []exercises.ExpectedOutputFile{t05blocksF},
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "14-ext2-blkiter",
		AuxSrc:    auxSrc,
		Testcases: []exercises.Testcase{t00, t01, t02, t03, t04, t05},
	})
}
