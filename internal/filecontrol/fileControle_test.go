package filecontrol

import (
	"os"
	"testing"
)

// The tests are going to take some thought.  I can probably test recurseDirTest
// by itself, but LocateFilesTest and TargetNameGenTest depend on each other too
// much to test them in isolation.

func TestLocateFiles(t *testing.T) {
	// Nothing yet.
}

func TestRecurseDir(t *testing.T) {
	// Nothing yet.

	foundFiles := make([]foundFileStr, 0)
	home := os.Getenv("HOME")
	fullPath := home + "/go/src/github.com/cardSlurp/internal/filecontrol/testData/source"
	debugMode := false
	err := recurseDir(fullPath, &foundFiles, &debugMode)
	if err != nil {
		t.Fail()
	}
}

func TestTargetNameGen(t *testing.T) {
	// Nothing yet.
}
