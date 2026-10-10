package exercises

import (
	"embed"
	"io/fs"

	"tests-runner/util/util/errutil"
)

var (
	//go:embed src/stdlib/*
	__auxSrcStdlib embed.FS

	AuxSrcStdlib fs.FS
)

func init() {
	var err error
	AuxSrcStdlib, err = fs.Sub(__auxSrcStdlib, "src/stdlib")
	errutil.FatalIf(err)
}
