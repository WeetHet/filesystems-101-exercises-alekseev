package cond

// If works as C ternary (if COND ? X : Y) operator
//
// NOTE, in some cases it may hit performance or even cause to side effects:
// 1. There is no guarantee operator is gonna be inlined
// 2. All the operator values ARE always pre-calculated, i.e. all function calls will be performed before checking the condition.
func If[T any](cond bool, v1, v2 T) T {
	if cond {
		return v1
	}
	return v2
}
