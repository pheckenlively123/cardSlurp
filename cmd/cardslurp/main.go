package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pheckenlively123/cardSlurp/cmd/cardslurp/internal/filecontrol"

	"github.com/pheckenlively123/cardSlurp/cmd/cardslurp/internal/commandline"
)

func main() {

	// Get command line options.
	opts := commandline.GetOpts()

	// Build the channel the other go routines will use to get the
	// target filenames.
	getTargetQueue := make(chan filecontrol.GetFileNameMsg)

	go filecontrol.TargetNameGen(getTargetQueue, opts.TargetDir, opts.TransBuff, opts.DebugMode)

	targLeafList, err1 := ioutil.ReadDir(opts.MountDir)
	if err1 != nil {
		panic("Error reading mountDir.\n")
	}

	foundCount := 0
	doneQueue := make(chan filecontrol.FinishMsg)

	for x := range targLeafList {

		leaf := targLeafList[x]

		if strings.Contains(leaf.Name(), opts.SearchStr) {

			fullPath := opts.MountDir + "/" + leaf.Name()

			fmt.Printf("Found match: %s\n", fullPath)

			// Spawn a thread to offload each card at the
			// same time.
			go filecontrol.LocateFiles(fullPath, doneQueue, getTargetQueue, opts.TransBuff, opts.DebugMode)
			foundCount++
		}
	}

	summary := make([]filecontrol.FinishMsg, 0)

	// Get results from the worker threads.
	for i := 0; i < foundCount; i++ {
		finishResult := <-doneQueue
		if finishResult.MajorErr != nil {
			panic("Major error locating and copying files")
		}
		summary = append(summary, finishResult)
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
