package utils

import (
	"fmt"
	"runtime"
	"strings"
)

// GetFileAndLoC returns the file path and line of code with skip being the number of stack frames to skip
func GetFileAndLoC(skip int) string {
	_, filepath, line, _ := runtime.Caller(1 + skip)

	// trim to only after watch-party
	if i := strings.LastIndex(filepath, "watch-party"); i != -1 {
		filepath = filepath[i:]
	}

	return fmt.Sprintf(
		"%s:%d",
		filepath,
		line,
	)
}
