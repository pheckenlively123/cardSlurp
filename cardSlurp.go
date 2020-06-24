package main

import (
	"cardSlurp/filecontrol"
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
)

var targetDir = flag.String("targetDir", "",
	"Target directory for the copied files.")
var mountDir = flag.String("mountDir", "",
	"Directory where cards are mounted.")
var searchStr = flag.String("searchStr", "",
	"String to distinguish cards from other mounted media in mountDir.")
var debugMode = flag.Bool("debugMode", false,
	"Print extra debug information.")
var transBuff = flag.Int("transBuff", 8192,
	"Transfer buffer size.")

func init() {
	flag.Parse()

	if *targetDir == "" {
		flag.PrintDefaults()
		panic("Missing -targetDir\n")
	}

	if *mountDir == "" {
		flag.PrintDefaults()
		panic("Missing -mountDir\n")
	}

	if *searchStr == "" {
		flag.PrintDefaults()
		panic("Missing -searchStr\n")
	}
}

func main() {

	// Build the channel the other go routines will use to get the
	// target filenames.
	getTargetQueue := make(chan filecontrol.GetFileNameMsg)

	go filecontrol.TargetNameGen(getTargetQueue, targetDir, transBuff, debugMode)

	targLeafList, err1 := ioutil.ReadDir(*mountDir)
	if err1 != nil {
		panic("Error reading mountDir.\n")
	}

	foundCount := 0
	doneQueue := make(chan filecontrol.FinishMsg)

	for x := range targLeafList {

		leaf := targLeafList[x]

		if strings.Contains(leaf.Name(), *searchStr) {

			fullPath := *mountDir + "/" + leaf.Name()

			fmt.Printf("Found match: %s\n", fullPath)

			// Spawn a thread to offload each card at the
			// same time.
			go filecontrol.LocateFiles(fullPath, doneQueue, getTargetQueue, transBuff, debugMode)
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
