package cardfileutil

import (
	"errors"
	"io"
	"os"
)

// IsFileSame - Do a byte by byte comparison of the two files.
func IsFileSame(thingOne string, thingTwo string, transBuff int) (bool, error) {

	from, err := os.Open(thingOne)
	if err != nil {
		return false, errors.New("Error opening: " + thingOne)
	}
	defer from.Close()

	to, err := os.Open(thingTwo)
	if err != nil {
		return false, errors.New("Error opening: " + thingTwo)
	}
	defer to.Close()

	nibFrom := make([]byte, transBuff)
	nibTo := make([]byte, transBuff)

	for {
		readFrom, errFrom := from.Read(nibFrom)
		readTo, errTo := to.Read(nibTo)

		if (errFrom == io.EOF) && (errTo == io.EOF) {
			break
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
		if errFrom == io.EOF {
			return false, nil
		}

		// One of the files finished before the other.
		if errTo == io.EOF {
			return false, nil
		}

		// Process any weird errors.
		if errFrom != nil {
			return false, errFrom
		}

		// Process any weird errors.
		if errTo != nil {
			return false, errTo
		}
	}

	return true, nil
}

// NibbleCopy - Copy one file to another a nibble at a time.
func NibbleCopy(thingOne string, thingTwo string, transBuff int) (bool, error) {

	from, err := os.Open(thingOne)
	if err != nil {
		return false, errors.New("Error opening: " + thingOne)
	}
	defer from.Close()

	to, err := os.OpenFile(thingTwo, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return false, errors.New("Error opening: " + thingTwo)
	}
	defer to.Close()

	nibble := make([]byte, transBuff)

	for {

		// I think I need to be checking for io.EOF in errFrom below.
		byteRead, errFrom := from.Read(nibble)
		if (byteRead == 0) && (errFrom == io.EOF) {
			// We appear to be done.
			return true, nil
		}
		if errFrom != nil {
			// Some other error must have happened.
			return false, errFrom
		}

		// Write the last block of bytes one at a time.
		if byteRead < transBuff {
			tag := make([]byte, byteRead)

			for i := 0; i < byteRead; i++ {
				tag[i] = nibble[i]
			}

			_, errTag := to.Write(tag)
			if errTag != nil {
				return false, errTag
			}
			break
		}

		_, errTo := to.Write(nibble)
		if errTo != nil {
			return false, errTo
		}

		if (errFrom == io.EOF) || (byteRead < transBuff) {
			break
		}
	}

	return true, nil
}
