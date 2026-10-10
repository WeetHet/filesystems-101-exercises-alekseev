package exercises

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"syscall"

	"tests-runner/util/errors"
)

type VerifierFunc func(file io.Reader) (r *TestResult, err error)

type VerifierLinesEqual struct {
	Expected []string
}

func LinesEqual(expected ...string) *VerifierLinesEqual {
	lines := make([]string, 0, len(expected))
	for _, s := range expected {
		s = strings.TrimSuffix(s, "\n")
		s = strings.TrimPrefix(s, "\n")
		if s != "" {
			lines = append(lines, s)
		}
	}
	return &VerifierLinesEqual{Expected: lines}
}

func LinesEqualS(expected string) *VerifierLinesEqual {
	allLines := strings.Split(expected, "\n")
	lines := make([]string, 0, len(allLines))
	for _, s := range allLines {
		if s != "" {
			lines = append(lines, s)
		}
	}
	return &VerifierLinesEqual{Expected: lines}
}

func (v *VerifierLinesEqual) Verify(file io.Reader) (r *TestResult, err error) {
	expected := v.Expected

	scanner := bufio.NewScanner(file)
	n := 0
	for scanner.Scan() {
		if len(expected) == 0 {
			return WrongAnswer("too many output lines"), nil
		}
		if scanner.Text() != expected[0] {
			r = WrongAnswer(fmt.Sprintf("line %d differs: got %q, expected %q",
				n, limitStr(scanner.Text(), 32), limitStr(expected[0], 32)))
			return r, nil
		}

		expected = expected[1:]
		n++
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	if len(expected) > 0 {
		return WrongAnswer("too few output lines"), nil
	}
	return CorrectAnswer(), nil
}

func (f *ExpectedOutputFile) Verify(outputDir string) (r *TestResult, err error) {
	file, err := os.Open(path.Join(outputDir, f.Name))
	if errors.Is(err, syscall.ENOENT) {
		return NoAnswer(f.IfMissing), nil
	}
	if err != nil {
		return r, err
	}
	defer file.Close()

	return f.Verifier(file)
}

func limitStr(x string, n int) string {
	if len(x) <= n {
		return x
	}
	m := len(x) - n
	return fmt.Sprintf("%s...(%d bytes more)", x[:n], m)

}

func BytesEqual(expected []byte) VerifierFunc {
	return func(file io.Reader) (r *TestResult, err error) {
		b := make([]byte, len(expected)+1)
		n, err := file.Read(b)
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		if n != len(expected) {
			return WrongAnswer("file length does not match"), nil
		}
		b = b[:n]
		if bytes.Compare(b, expected) != 0 {
			return WrongAnswer("file content differs"), nil
		}
		return CorrectAnswer(), nil
	}
}
