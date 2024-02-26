package utils

import (
	"fmt"
	"os"
	"strconv"
)

func Log(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
}

func StringToUint(s string) uint {
	i, err := strconv.Atoi(s)
	if err != nil {
		Log("convert %q to uint: %w", s, err)
		os.Exit(2)
	}
	return uint(i)
}
