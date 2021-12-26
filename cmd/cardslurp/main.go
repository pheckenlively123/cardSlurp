package main

import (
	"fmt"

	"github.com/pheckenlively123/cardSlurp/internal/commandline"
	"github.com/pheckenlively123/cardSlurp/internal/filecontrol"
)

func main() {

	// Get command line options.
	opts, err := commandline.GetOpts()
	if err != nil {
		// No point in continuing.
		panic("error processing command line arguments: " + err.Error())
	}

	// Build the channel the other go routines will use to get the
	// target filenames.
	getTargetQueue := make(chan filecontrol.GetFileNameMsg)

	go filecontrol.TargetNameGen(getTargetQueue, opts.TargetDir, opts.TransBuff, opts.DebugMode)

	doneQueue := make(chan filecontrol.FinishMsg)

	for _, mp := range opts.MountList {
		go filecontrol.LocateFiles(mp, doneQueue, getTargetQueue, opts.TransBuff, opts.DebugMode)
	}

	summary := make([]filecontrol.FinishMsg, 0)

	// Get results from the worker threads.
	for i := 0; i < len(opts.MountList); i++ {
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
