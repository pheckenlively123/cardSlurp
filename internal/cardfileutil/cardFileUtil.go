package cardfileutil

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// Making the functions in this package into methods doesn't really add
// much to the functionality.  The big benefit is that we can mock this
// with an interface elsewhere.
type CardFileUtil struct {
	transBufferSize    uint64
	verificationPasses uint64
}

func NewCardFileUtil(transBufferSize uint64,
	verificationPasses uint64) *CardFileUtil {
	return &CardFileUtil{
		transBufferSize:    transBufferSize,
		verificationPasses: verificationPasses,
	}
}

// IsFileSame - Do a byte by byte comparison of the two files.
func (c *CardFileUtil) IsFileSame(fromFile string, toFile string) (bool, error) {

	// While it is tempting to

	from, err := os.Open(fromFile)
	if err != nil {
		return false, errors.New("Error opening: " + fromFile)
	}
	defer from.Close()

	to, err := os.Open(toFile)
	if err != nil {
		return false, errors.New("Error opening: " + toFile)
	}
	defer to.Close()

	nibFrom := make([]byte, c.transBufferSize)
	nibTo := make([]byte, c.transBufferSize)

	for i := 0; i < int(c.verificationPasses); i++ {

	VERIFYLOOP:
		for {
			readFrom, errFrom := from.Read(nibFrom)
			readTo, errTo := to.Read(nibTo)

			if errors.Is(errFrom, io.EOF) && errors.Is(errTo, io.EOF) {
				// The two files finished at the same time, which is
				// what we want, if they are the same.
				break VERIFYLOOP
			}

			if readFrom != readTo {
				return false, nil
			}

			for x := range nibFrom {
				if nibFrom[x] != nibTo[x] {
					return false, nil
				}
			}

			// One of the files finished before the other.
			if errors.Is(errFrom, io.EOF) {
				return false, nil
			}

			// One of the files finished before the other.
			if errors.Is(errTo, io.EOF) {
				return false, nil
			}

			// Process any weird errors.
			if errFrom != nil {
				return false, fmt.Errorf(
					"unexpected error reading from from file during validation: %w", errFrom)
			}

			// Process any weird errors.
			if errTo != nil {
				return false, fmt.Errorf(
					"unexpected error reading from to during validation: %w", errTo)
			}
		}
	}

	return true, nil
}

// NibbleCopy - Copy one file to another a nibble at a time.
func (c *CardFileUtil) CopyCardFile(fromFile string, toFile string) (bool, error) {

	from, err := os.Open(fromFile)
	if err != nil {
		return false, errors.New("Error opening: " + fromFile)
	}
	defer from.Close()

	to, err := os.OpenFile(toFile, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return false, errors.New("Error opening: " + toFile)
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return false, fmt.Errorf("error copying card file: %w", err)
	}

	return true, nil
}
