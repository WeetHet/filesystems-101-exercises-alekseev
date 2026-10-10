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

	//go:embed "testdata/ls"
	t00ls  string
	t00lsF = exercises.ExpectedOutputFile{
		Name:      "ls",
		IfMissing: "solution never created the output file?",
		Verifier:  exercises.LinesEqualS(t00ls).Verify,
	}
	t00 = exercises.Testcase{
		Name:            "00-ls",
		Args:            []string{"test", "00-ls"},
		WithoutValgrind: true,
		ExpectedOutputs: []exercises.ExpectedOutputFile{t00lsF},
	}

	//go:embed "testdata/dump-hello"
	t01dump  []byte
	t01dumpF = exercises.ExpectedOutputFile{
		Name:      "dump-hello",
		IfMissing: "solution never created the output file?",
		Verifier:  exercises.BytesEqual(t01dump),
	}
	t01 = exercises.Testcase{
		Name:            "01-read-helloworld",
		Args:            []string{"test", "01-read-helloworld"},
		WithoutValgrind: true,
		ExpectedOutputs: []exercises.ExpectedOutputFile{t01dumpF},
	}

	//go:embed "testdata/dump-direct"
	t02dumpD []byte
	//go:embed "testdata/dump-singly-indirect"
	t02dumpSI []byte
	//go:embed "testdata/dump-doubly-indirect"
	t02dumpDI []byte
	t02dumpDF = exercises.ExpectedOutputFile{
		Name:      "dump-direct",
		IfMissing: "solution never created the output file?",
		Verifier:  exercises.BytesEqual(t02dumpD),
	}
	t02dumpSIF = exercises.ExpectedOutputFile{
		Name:      "dump-singly-indirect",
		IfMissing: "solution never created the output file?",
		Verifier:  exercises.BytesEqual(t02dumpSI),
	}
	t02dumpDIF = exercises.ExpectedOutputFile{
		Name:      "dump-doubly-indirect",
		IfMissing: "solution never created the output file?",
		Verifier:  exercises.BytesEqual(t02dumpDI),
	}
	t02 = exercises.Testcase{
		Name:            "02-read-2ind",
		Args:            []string{"test", "02-read-2ind"},
		WithoutValgrind: true,
		ExpectedOutputs: []exercises.ExpectedOutputFile{t02dumpDF, t02dumpSIF, t02dumpDIF},
	}

	t03 = exercises.Testcase{
		Name:            "03-enoent",
		Args:            []string{"test", "03-enoent"},
		WithoutValgrind: true,
	}
	t04 = exercises.Testcase{
		Name:            "04-enotdir",
		Args:            []string{"test", "04-enotdir"},
		WithoutValgrind: true,
	}
)

func init() {
	auxSrc, err := fs.Sub(__auxSrc, "src")
	errutil.FatalIf(err)

	exercises.RegisterExercise(exercises.Exercise{
		Name:      "10-ext2-fuse",
		AuxSrc:    auxSrc,
		Testcases: []exercises.Testcase{t00, t01, t02, t03, t04},
	})
}
