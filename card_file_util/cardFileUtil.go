package card_file_util

import (
	"errors"
	"io"
	"os"
)

// Do a byte by byte comparison of the two files.
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

// Copy the file located at thingOne to location thingTwo.
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
		byteRead, errFrom := from.Read(nibble)

		if errFrom != nil {
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
