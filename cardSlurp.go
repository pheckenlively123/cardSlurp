package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type finishMsg struct {
	fullPath string
	skipped  int
	copied   int
	errors   []string
}

// Not thrilled about passing the entire file through the channel, but
// I can't think of a way for the target name goroutine to determine
// if a target file is the same otherwise.
type getFileNameMsg struct {
	leafName string
	leafData []byte
	callback chan returnFileNameMsg
}

type returnFileNameMsg struct {
	writeLeafName string
	skipFlag      bool
}

type foundFileStr struct {
	fullPath string
	leafName string
	leafMode os.FileMode
}

var targetDir = flag.String("targetDir", "",
	"Target directory for the copied files.")
var mountDir = flag.String("mountDir", "",
	"Directory where cards are mounted.")
var searchStr = flag.String("searchStr", "",
	"String to distinguish cards from other mounted media in mountDir.")
var debugMode = flag.Bool("debugMode", false,
	"Print extra debug information.")

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
	getTargetQueue := make(chan getFileNameMsg)

	go targetNameGen(getTargetQueue)

	targLeafList, err1 := ioutil.ReadDir(*mountDir)
	if err1 != nil {
		panic("Error reading mountDir.\n")
	}

	foundCount := 0
	doneQueue := make(chan finishMsg)

	for x := range targLeafList {

		leaf := targLeafList[x]

		if strings.Contains(leaf.Name(), *searchStr) {

			fullPath := *mountDir + "/" + leaf.Name()

			fmt.Printf("Found match: %s\n", fullPath)

			// Spawn a thread to extract each card at the
			// same time.
			go locateFiles(fullPath, doneQueue, getTargetQueue)
			foundCount++
		}
	}

	summary := make([]finishMsg, 0)

	// Get results from the worker threads.
	for i := 0; i < foundCount; i++ {
		finishResult := <-doneQueue
		summary = append(summary, finishResult)
	}


	errorFlag := false
	
	// Print the summary results.
	for x := range summary {

		r := summary[x]

		fmt.Printf("Card path: %s\n", r.fullPath)
		fmt.Printf("Skipped: %d - Copied: %d\n", r.skipped, r.copied)

		if len(r.errors) == 0 {
			fmt.Printf("(No errors.)\n")
		} else {
			fmt.Printf("*** ERRORS ***\n")
			errorFlag = true

			for y := range r.errors {
				fmt.Printf("%s\n", r.errors[y])
			}
		}
	}

	if errorFlag {
		fmt.Printf("*** Warning - Errors Found ***\n")
	}
}

func locateFiles(fullPath string, doneMsg chan finishMsg,
	getTargetQueue chan getFileNameMsg) {

	retVal := new(finishMsg)
	retVal.fullPath = fullPath

	foundFiles := make([]foundFileStr, 0)

	recurseDir(fullPath, &foundFiles)

	for x := range foundFiles {

		f := foundFiles[x]

		sourceFile := f.fullPath + "/" + f.leafName
		sourceData, err1 := ioutil.ReadFile(sourceFile)
		if err1 != nil {
			retVal.errors = append(retVal.errors,
				"Error reading "+f.fullPath+"/"+f.leafName)
			continue
		}

		targFileMsg := new(getFileNameMsg)
		targFileMsg.callback = make(chan returnFileNameMsg)
		targFileMsg.leafName = f.leafName
		targFileMsg.leafData = sourceData

		getTargetQueue <- *targFileMsg

		callBackMsg := <-targFileMsg.callback

		if *debugMode {
			fmt.Printf("Using %s for write name.\n",
				callBackMsg.writeLeafName)
		}

		if callBackMsg.skipFlag {
			retVal.skipped++
			fmt.Printf("Skipping, because it is already saved in the target: %s\n", f.leafName)
			continue
		}

		// If we got this far, then we have a save name, and
		// the file isn't already saved into the target
		// directory.
		// Make sure the target
		targFile := callBackMsg.writeLeafName
		err2 := ioutil.WriteFile(targFile, sourceData, f.leafMode)
		if err2 != nil {

			retVal.errors = append(retVal.errors,
				"Error writing: "+targFile)
			continue
		}

		// If we got this far, we have written the image to
		// disk.  Time to make sure what we wrote matches what
		// was read from the card.
		readBack, err3 := ioutil.ReadFile(targFile)
		if err3 != nil {

			retVal.errors = append(retVal.errors,
				"Error reading back: "+targFile)
			continue

		}

		if isFileSame(sourceData, readBack) {
			retVal.copied++
			fmt.Printf("%s/%s - Done\n", f.fullPath, f.leafName)
		} else {
			retVal.errors = append(retVal.errors,
				"File does not match what was written: %s",
				targFile)
		}

	}

	// Let the main routine know we are done.
	doneMsg <- *retVal
}

func recurseDir(fullPath string, foundFiles *[]foundFileStr) {

	fmt.Printf("Recursing: %s\n", fullPath)

	leafList, err1 := ioutil.ReadDir(fullPath)
	if err1 != nil {
		panic(err1)
	}

	for x := range leafList {
		leaf := leafList[x]

		switch mode := leaf.Mode(); {
		case mode.IsRegular():

			foundRec := new(foundFileStr)
			foundRec.fullPath = fullPath
			foundRec.leafName = leaf.Name()
			foundRec.leafMode = leaf.Mode()

			*foundFiles = append(*foundFiles, *foundRec)

			if *debugMode {
				fmt.Printf("Found: %s/%s\n", fullPath,
					leaf.Name())
			}
		case mode.IsDir():
			newPath := fullPath + "/" + leaf.Name()
			recurseDir(newPath, foundFiles)
		case mode&os.ModeSymlink != 0:
			fmt.Printf("Symlink: %s\n", leaf.Name())
			panic("Do not know how to process symlinks.\n")
		case mode&os.ModeNamedPipe != 0:
			fmt.Printf("Named pipe: %s\n", leaf.Name())
			panic("Do not know how to process pipes.\n")
		default:
			fmt.Printf("Got unknown file type: %s\n",
				leaf.Name())
			panic("Do not know how to process unknown files.\n")
		}
	}
}

// Wait for the card processors to request filenames.  Perhaps look at
// adding some directions to my channel below, just to make it obvious
// what is going on, and to provide some type safety.

// This approach is careful, but not full proof.  If something else is
// writing to the target directory, this program could still overwrite
// it.  However, it will step around anything that is already there.
// It is also aware of any files it has blessed for writing.  A
// foolproof way to make sure we have a unique name would be to use
// the system open with excusive and create flags.  That approach
// would probably be portable to MacOS, because it is based on BSD
// UNIX.  I don't think it would be portable to Windoze.

//I also considered using uuids for generating unique names.  It would
//have worked without all the fun of channeling all the threads
//through the goroutine below.  The downside would have been filenames
//that differed significantly from the names on the cards.
func targetNameGen(getTargetQueue chan getFileNameMsg) {

	// Need to track what has been given for file names, so we can
	// make sure there are no conflicts.
	targMap := make(map[string]bool)

	for {
		// This thread blocks here until it gets a request on
		// the channel.
		request := <-getTargetQueue

		if *debugMode {
			fmt.Printf("Got target name request for: %s\n",
				request.leafName)
		}

		callbackMsg := new(returnFileNameMsg)

		var tryName string

		// Loop until we have a valid target name
		for i := 0; i < 999999; i++ {

			// Failsafe
			if i == 999998 {
				panic("Error renaming:" + request.leafName)
			}

			if i == 0 {

				tryName = *targetDir + "/" + request.leafName

			} else {
				leafParts := strings.Split(request.leafName, ".")
				leafStub := ""
				leafExt := leafParts[len(leafParts)-1]

				for i := 0; i < (len(leafParts) - 1); i++ {
					if leafStub == "" {
						leafStub = leafParts[i]
					} else {
						leafStub = leafStub + "." + leafParts[i]
					}
				}

				tryName = fmt.Sprintf("%s/%s%s%d.%s",
					*targetDir, leafStub, "-", i, leafExt)

				if *debugMode {
					fmt.Printf("Trying: %s\n", tryName)
				}
			}

			if targMap[tryName] {
				// This name has already been used.
				// Try again.
				continue
			}

			if _, err := os.Stat(tryName); os.IsNotExist(err) {
				// tryName does not exist.  We should
				// be OK to write

				callbackMsg.writeLeafName = tryName
				targMap[tryName] = true
				break
			} else {
				// File with the same name is already
				// there.  Check to see if the file is
				// the same as the one I'm trying to
				// write.

				tryData, err2 := ioutil.ReadFile(tryName)
				if err2 != nil {
					panic("Failed to read: " + tryName)
				}

				if isFileSame(tryData, request.leafData) {
					// Tell the caller to skip the write.
					callbackMsg.writeLeafName = tryName
					callbackMsg.skipFlag = true
					targMap[tryName] = true
					break
				} else {
					// This filename is taken.  Find another.
					continue
				}
			}
		}

		// The send back the result.
		request.callback <- *callbackMsg
	}
}

func isFileSame(thingOne []byte, thingTwo []byte) bool {

	if len(thingOne) != len(thingTwo) {
		return false
	}

	for x := range thingOne {
		if thingOne[x] != thingTwo[x] {
			return false
		}
	}

	return true
}
