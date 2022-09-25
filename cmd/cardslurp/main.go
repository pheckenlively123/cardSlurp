package main

import (
	"fmt"

	"github.com/pheckenlively123/cardSlurp/cmd/cardslurp/internal/filecontrol"

	"github.com/pheckenlively123/cardSlurp/cmd/cardslurp/internal/commandline"
)

func main() {

	// Get command line options.
	opts, err := commandline.GetOpts()
	if err != nil {
		// No point in continuing.
		panic("error processing command line arguments: " + err.Error())
	}

	foundCount := 0
	doneQueue := make(chan filecontrol.FinishMsg)

	targetNameManager := filecontrol.NewTargetNameGenManager(opts.TargetDir)

	for _, mountDir := range opts.MountList {

		// Spawn a thread to offload each card at the
		// same time.
		go filecontrol.LocateFiles(mountDir, doneQueue, targetNameManager, opts.TransBuff,
			opts.DebugMode)
		foundCount++
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
