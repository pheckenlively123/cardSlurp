package cardfileutil

import (
	"fmt"
	"testing"
)

func TestIsFileSame(t *testing.T) {

	maxTransBuff := 8192

	// Torture test for boundary conditions.  :-)
	for transBuff := 1; transBuff <= maxTransBuff; transBuff++ {

		sameStat, err := IsFileSame("testData/same_a.txt", "testData/same_b.txt", transBuff)
		if err != nil {
			fmt.Print("Error calling IsFileSame: " + err.Error() + "\n")
			t.Fail()
		}

		if !sameStat {
			t.Error("Files were the same, but they tested as different.\n")
		}

		diffStat, err := IsFileSame("testData/same_a.txt", "testData/diff_b.txt", transBuff)
		if err != nil {
			fmt.Print("Error calling IsFileSame: " + err.Error() + "\n")
		}

		if diffStat {
			t.Error("Files were different, but they tested as the same.\n")
		}
	}
}

func TestNibbleCopy(t *testing.T) {

	maxTransBuff := 8192

	for transBuff := 1; transBuff <= maxTransBuff; transBuff++ {

		nibStat, err := NibbleCopy("testData/same_a.txt", "testData/victim.txt", transBuff)
		if err != nil {
			print("Error calling NibbleCopy: " + err.Error() + "\n")
			t.Fail()
		}

		if !nibStat {
			t.Errorf("Nibble copy returned false: transBuff == %d\n", transBuff)
		}

		sameStat, err := IsFileSame("testData/same_a.txt", "testData/victim.txt", transBuff)
		if err != nil {
			print("Error calling IsFileSame: " + err.Error() + "\n")
			t.Fail()
		}

		if !sameStat {
			t.Error("IsFileSame said that file copied by NibbleCopy differed.\n")
		}
	}
}
