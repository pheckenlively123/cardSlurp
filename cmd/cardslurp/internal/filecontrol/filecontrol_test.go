package filecontrol

import (
	"fmt"
	"os"
	"testing"
)

// The tests are going to take some thought.  I can probably test recurseDirTest
// by itself, but LocateFilesTest and TargetNameGenTest depend on each other too
// much to test them in isolation.

func TestRecurseDir(t *testing.T) {

	// Base the path of the test source on the home directory variable.  That
	// way, this test should work with other developers too.  :-P
	homedir := os.Getenv("HOME")

	foundFiles := make([]CardSlurpWork, 0)
	fullPath := homedir + "/go/src/cardSlurp/cmd/cardslurp/internal/filecontrol/testData/source"
	debugMode := false
	err := recurseDir(fullPath, &foundFiles, &debugMode)
	if err != nil {
		t.Fatal("recurseDir() returned an error")
	}
}

// This test mimics the behavior of the main application.
// The main purpose is to exersize everything in dlv.
func TestNewWorkerPool(t *testing.T) {

	homedir := os.Getenv("HOME")

	testDir := homedir + "/go/src/github.com/cardSlurp/cmd/cardslurp/internal/filecontrol/testData"

	cardA := testDir + "/source/A"
	cardB := testDir + "/source/B"
	cardC := testDir + "/source/C"
	cardD := testDir + "/source/D"

	targetDir := testDir + "/target"

	// Try to remove the targetdir, in case a test died in the middle.
	// Then make it.
	_ = os.RemoveAll(targetDir)
	err := os.MkdirAll(targetDir, 0777)
	if err != nil {
		t.Fatal("error making targetdir: " + err.Error())
	}

	nameOracle, err := NewTargetNameGenManager(targetDir, 2)
	if err != nil {
		t.Fatal("error making name oracle: " + err.Error())
	}

	workerPool := NewWorkerPool(15, nameOracle, 2, true, 5)

	err = OrchestrateLocate([]string{cardA, cardB, cardC, cardD},
		workerPool, true)
	if err != nil {
		t.Fatal("unexpected from OrchestrateLocate: " + err.Error())
	}

	finalResults, err := workerPool.ParallelFileCopy()
	if err != nil {
		t.Fatal("unexpected error from parallel file copy: " + err.Error())
	}

	// Print the summary results.
	fmt.Printf("Skipped: %d - Copied: %d\n", finalResults.Skipped,
		finalResults.Copied)
	if len(finalResults.MinorErrs) == 0 {
		fmt.Printf("(No errors.)\n")
	} else {
		fmt.Printf("*** ERRORS ***\n")

		for y := range finalResults.MinorErrs {
			fmt.Printf("%s\n", finalResults.MinorErrs[y])
		}
	}
}
