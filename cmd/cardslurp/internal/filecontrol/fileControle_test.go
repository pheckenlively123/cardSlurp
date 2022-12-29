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

	// Base the path of the test source on the home directory variable.  That
	// way, this test should work with other developers too.  :-P
	homedir := os.Getenv("HOME")

	foundFiles := make([]foundFile, 0)
	fullPath := homedir + "/go/src/cardSlurp/cmd/cardslurp/internal/filecontrol/testData/source"
	debugMode := false
	err := recurseDir(fullPath, &foundFiles, &debugMode)
	if err != nil {
		t.Fatal("recurseDir() returned an error")
	}
}

func TestTargetNameGen(t *testing.T) {
	// Nothing yet.
}
