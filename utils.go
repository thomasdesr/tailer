package tailer

import "path/filepath"

func pathEqual(lhs, rhs string) bool {
	var err error
	lhs, err = filepath.Abs(lhs)
	if err != nil {
		return false
	}
	rhs, err = filepath.Abs(rhs)
	if err != nil {
		return false
	}
	return lhs == rhs
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
