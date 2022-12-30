package cardfileutil

import (
	"fmt"
	"io"
	"os"
)

// IsFileSame - Do a byte by byte comparison of the two files.
// Iterations are the number of times we should compare the two files.
func IsFileSame(fromFile string, toFile string,
	iterations uint64) (bool, error) {

	for i := 0; i < int(iterations); i++ {

		fromBytes, err := os.ReadFile(fromFile)
		if err != nil {
			return false, fmt.Errorf("error reading from file: %w", err)
		}

		toBytes, err := os.ReadFile(toFile)
		if err != nil {
			return false, fmt.Errorf("error reading to file: %w", err)
		}

		if len(fromBytes) != len(toBytes) {
			return false, nil
		}

		for i := 0; i < len(fromBytes); i++ {
			if fromBytes[i] != toBytes[i] {
				return false, nil
			}
		}
	}

	return true, nil
}

// FileCopy - Copy one file to another.
func FileCopy(fromFile string, toFile string) (bool, error) {

	from, err := os.Open(fromFile)
	if err != nil {
		return false, fmt.Errorf("error opening from file: %w", err)
	}
	defer from.Close()

	to, err := os.OpenFile(toFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return false, fmt.Errorf("error opening to file: %w", err)
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return false, fmt.Errorf("error copying from to to: %w", err)
	}

	return true, nil
}
