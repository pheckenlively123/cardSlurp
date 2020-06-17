package card_file_util

import "testing"

func TestIsFileSame(t *testing.T) {

	transBuff := 8192

	sameStat, err := IsFileSame("testData/same_a.txt", "testData/same_b.txt", transBuff)
	if err != nil {
		print("Error calling IsFileSame: " + err.Error() + "\n")
		t.Fail()
	}

	if !sameStat {
		t.Error("Files were the same, but they tested as different.\n")
	}

	diffStat, err := IsFileSame("testData/same_a.txt", "testData/diff_b.txt", transBuff)
	if err != nil {
		print("Error calling IsFileSame: " + err.Error() + "\n")
	}

	if diffStat {
		t.Error("Files were different, but they tested as the same.\n")
	}
}

func TestNibbleCopy(t *testing.T) {

	transBuff := 8192

	nibStat, err := NibbleCopy("testData/same_a.txt", "testData/victim.txt", transBuff)
	if err != nil {
		print("Error calling NibbleCopy: " + err.Error() + "\n")
		t.Fail()
	}

	if !nibStat {
		t.Error("Nibble copy returned false.\n")
	}
}
