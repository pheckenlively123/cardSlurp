package cardfileutil

import (
	"fmt"
	"testing"
)

func TestIsFileSame(t *testing.T) {

	maxTransBuff := 8192

	// Torture test for boundary conditions.  :-)
	for transBuff := 1; transBuff <= maxTransBuff; transBuff++ {

		cfu := NewCardFileUtil(uint64(transBuff), 3)
		sameStat, err := cfu.IsFileSame("testData/same_a.txt", "testData/same_b.txt")
		if err != nil {
			fmt.Print("Error calling IsFileSame: " + err.Error() + "\n")
			t.Fail()
		}

		if !sameStat {
			t.Error("Files were the same, but they tested as different.\n")
		}

		diffStat, err := cfu.IsFileSame("testData/same_a.txt", "testData/diff_b.txt")
		if err != nil {
			fmt.Print("Error calling IsFileSame: " + err.Error() + "\n")
		}

		if diffStat {
			t.Error("Files were different, but they tested as the same.\n")
		}
	}
}

func TestCopyCardFile(t *testing.T) {

	maxTransBuff := 8192

	for transBuff := 1; transBuff <= maxTransBuff; transBuff++ {

		cfu := NewCardFileUtil(uint64(transBuff), 3)

		nibStat, err := cfu.CardFileCopy("testData/same_a.txt", "testData/victim.txt")
		if err != nil {
			t.Fatal("Error calling NibbleCopy: " + err.Error())
		}

		if !nibStat {
			t.Fatalf("CopyCardFile returned false: transBuff == %d\n", transBuff)
		}

		sameStat, err := cfu.IsFileSame("testData/same_a.txt", "testData/victim.txt")
		if err != nil {
			t.Fatal("Error calling IsFileSame: " + err.Error())
		}

		if !sameStat {
			t.Error("IsFileSame said that file copied by NibbleCopy differed.\n")
		}
	}
}
