package file_control

import (
	"testing"
)

// The tests are going to take some thought.  I can probably test RecurseDirTest
// by itself, but LocateFilesTest and TargetNameGenTest depend on each other too
// much to test them in isolation.

func TestLocateFiles(t *testing.T) {
	// Nothing yet.
}

func TestRecurseDir(t *testing.T) {
	// Nothing yet.

	foundFiles := make([]FoundFileStr, 0)
	fullPath := "/home/pheckenl/go/src/cardSlurp/file_control/testData/source"
	debugMode := false
	err := RecurseDir(fullPath, &foundFiles, &debugMode)
	if err != nil {
		t.Fail()
	}
}

func TestTargetNameGen(t *testing.T) {
	// Nothing yet.
}
