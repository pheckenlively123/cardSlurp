package filecontrol

import (
	"context"
	"fmt"
	"os"
	"sync"
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

	testDir := homedir + "/go/src/cardSlurp/cmd/cardslurp/internal/filecontrol/testData"

	cardA := testDir + "/source/A"
	cardB := testDir + "/source/B"
	cardC := testDir + "/source/C"
	cardD := testDir + "/source/D"

	t.Log(cardA, cardB, cardC, cardD)

	targetDir := testDir + "/target"

	// Try to remove the targetdir, in case a test died in the middle.
	// Then make it.
	_ = os.RemoveAll(targetDir)
	err := os.MkdirAll(targetDir, 0777)
	if err != nil {
		t.Fatal("error making targetdir: " + err.Error())
	}

	nameManager, err := NewTargetNameGenManager(targetDir, 2)
	if err != nil {
		t.Fatal("error making name oracle: " + err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}

	workerPool := NewWorkerPool(ctx, 15, wg, nameManager, 2, true, 5)
	defer workerPool.Close()

	doneQueue := make(chan FinishMsg)

	go LocateFiles(cardA, doneQueue, nameManager, workerPool, true)
	go LocateFiles(cardB, doneQueue, nameManager, workerPool, true)
	go LocateFiles(cardC, doneQueue, nameManager, workerPool, true)
	go LocateFiles(cardD, doneQueue, nameManager, workerPool, true)

	foundCount := 4

	summary := make([]FinishMsg, 0)

	// Get results
	for i := 0; i < foundCount; i++ {
		finishMsg := <-doneQueue
		if finishMsg.MajorErr != nil {
			t.Fatal("got major error from one of the cards: " + finishMsg.MajorErr.Error())
		}
		summary = append(summary, finishMsg)
	}

	errorFlag := false

	// Print the summary results.
	for x := range summary {

		r := summary[x]

		fmt.Printf("Card path: %s\n", r.FullPath)
		fmt.Printf("Skipped: %d - Copied: %d\n", r.Skipped, r.Copied)

		if len(r.MinorErrs) == 0 {
			fmt.Printf("(No errors.)\n")
		} else {
			fmt.Printf("*** ERRORS ***\n")
			errorFlag = true

			for y := range r.MinorErrs {
				fmt.Printf("%s\n", r.MinorErrs[y])
			}
		}
	}

	if errorFlag {
		fmt.Printf("*** Warning - Errors Found ***\n")
	}
}
