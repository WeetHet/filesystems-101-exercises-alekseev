package exercises

import (
	"fmt"
)

func (t *TestResult) IsOk() bool {
	return t.Result == "ok"
}

func CorrectAnswer() *TestResult {
	return &TestResult{Result: "ok"}
}

func Killed(message string) *TestResult {
	return &TestResult{
		Result:  "killed",
		Message: message,
	}
}

func NonZeroExitCode(exitCode int, stderr string) *TestResult {
	return &TestResult{
		Result: fmt.Sprintf("non-zero exit code: %d", exitCode),
		Stderr: stderr,
	}
}

func VgDirty(vgOutput string) *TestResult {
	return &TestResult{
		Result:   "not vg-clean",
		VgOutput: vgOutput,
	}
}

func NoAnswer(message string) *TestResult {
	return &TestResult{
		Result:  "no answer",
		Message: message,
	}
}

func WrongAnswer(message string) *TestResult {
	return &TestResult{
		Result:  "wrong answer",
		Message: message,
	}
}
