package cardfileutil

import (
	"testing"
)

func TestIsFileSame(t *testing.T) {

	// Torture test for boundary conditions.  :-)
	for iterations := 1; iterations <= 5; iterations++ {

		sameStat, err := IsFileSame("testData/same_a.txt", "testData/same_b.txt", uint64(iterations))
		if err != nil {
			t.Fatal("Error calling IsFileSame: " + err.Error())
		}

		if !sameStat {
			t.Fatal("Files were the same, but they tested as different.\n")
		}

		diffStat, err := IsFileSame("testData/same_a.txt", "testData/diff_b.txt", uint64(iterations))
		if err != nil {
			t.Fatal("Error calling IsFileSame: " + err.Error())
		}

		if diffStat {
			t.Fatal("Files were different, but they tested as the same.\n")
		}
	}
}

func TestFileCopy(t *testing.T) {

	for iterations := 1; iterations <= 5; iterations++ {

		copyStatus, err := CardFileCopy("testData/same_a.txt", "testData/victim.txt")
		if err != nil {
			t.Fatal("Error calling FileCopy: " + err.Error())
		}

		if !copyStatus {
			t.Fatal("Nibble copy returned false")
		}

		sameStat, err := IsFileSame("testData/same_a.txt", "testData/victim.txt", uint64(iterations))
		if err != nil {
			t.Fatal("Error calling IsFileSame: " + err.Error())
		}

		if !sameStat {
			t.Fatal("IsFileSame said that file copied by NibbleCopy differed.")
		}
	}
}
