package filecontrol

import (
	"cardSlurp/cardfileutil"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// FinishMsg - passed back to main from the per card worker threads.
type FinishMsg struct {
	FullPath  string
	Skipped   int
	Copied    int
	MajorErr  error
	MinorErrs []string
}

// GetFileNameMsg - card worker threads use this to get unique names from the thread running TargetNameGen.
type GetFileNameMsg struct {
	LeafName string
	FullName string
	Callback chan returnFileNameMsg
}

type returnFileNameMsg struct {
	WriteLeafName string
	SkipFlag      bool
}

type foundFileStr struct {
	FullPath string
	LeafName string
	LeafMode os.FileMode
}

// LocateFiles - Main spins up a copy of this function for each of the cards we are offloading.
func LocateFiles(fullPath string, doneMsg chan FinishMsg, getTargetQueue chan GetFileNameMsg, transBuff int, debugMode bool) {

	rv := new(FinishMsg)
	rv.FullPath = fullPath

	foundFiles := make([]foundFileStr, 0)

	err := recurseDir(fullPath, &foundFiles, &debugMode)
	if err != nil {
		// Punch out early
		rv.MajorErr = err
		doneMsg <- *rv
		return
	}

	for x := range foundFiles {

		f := foundFiles[x]

		sourceFile := f.FullPath + "/" + f.LeafName

		targFileMsg := new(GetFileNameMsg)
		targFileMsg.Callback = make(chan returnFileNameMsg)
		targFileMsg.LeafName = f.LeafName
		targFileMsg.FullName = sourceFile

		getTargetQueue <- *targFileMsg

		callBackMsg := <-targFileMsg.Callback

		if debugMode {
			fmt.Printf("Using %s for write name.\n",
				callBackMsg.WriteLeafName)
		}

		if callBackMsg.SkipFlag {
			rv.Skipped++
			fmt.Printf("Skipping, because it is already saved in the target: %s\n", f.LeafName)
			continue
		}

		_, err := cardfileutil.NibbleCopy(sourceFile, callBackMsg.WriteLeafName, transBuff)
		if err != nil {
			rv.MinorErrs = append(rv.MinorErrs,
				"Error copying: "+sourceFile)
			fmt.Print("Error copying: " + sourceFile)
			continue
		}

		sameStat, err := cardfileutil.IsFileSame(sourceFile, callBackMsg.WriteLeafName, transBuff)
		if err != nil {
			rv.MinorErrs = append(rv.MinorErrs,
				"Error checking files are same: "+sourceFile)
			fmt.Print("Error verifying copy for: " + sourceFile)
			continue
		}

		if sameStat {
			rv.Copied++
			fmt.Printf("%s/%s - Done\n", f.FullPath, f.LeafName)
		} else {
			rv.MinorErrs = append(rv.MinorErrs,
				"file verification did not match for: "+sourceFile)
		}
	}

	// Let the main routine know we are done.
	doneMsg <- *rv
}

func recurseDir(fullPath string, foundFiles *[]foundFileStr, debugMode *bool) error {

	fmt.Printf("Recursing: %s\n", fullPath)

	leafList, err := ioutil.ReadDir(fullPath)
	if err != nil {
		return err
	}

	for x := range leafList {
		leaf := leafList[x]

		switch mode := leaf.Mode(); {
		case mode.IsRegular():

			foundRec := new(foundFileStr)
			foundRec.FullPath = fullPath
			foundRec.LeafName = leaf.Name()
			foundRec.LeafMode = leaf.Mode()

			*foundFiles = append(*foundFiles, *foundRec)

			if *debugMode {
				fmt.Printf("Found: %s/%s\n", fullPath,
					leaf.Name())
			}
		case mode.IsDir():
			newPath := fullPath + "/" + leaf.Name()
			recErr := recurseDir(newPath, foundFiles, debugMode)
			if recErr != nil {
				return recErr
			}
		case mode&os.ModeSymlink != 0:
			fmt.Printf("Symlink: %s\n", leaf.Name())
			fail := errors.New("We do not know how to process symlinks")
			return fail
		case mode&os.ModeNamedPipe != 0:
			fmt.Printf("Named pipe: %s\n", leaf.Name())
			fail := errors.New("Do not know how to process pipes")
			return fail
		default:
			fmt.Printf("Got unknown file type: %s\n", leaf.Name())
			fail := errors.New("Found unknown file type")
			return fail
		}
	}

	return nil
}

// Wait for the card processors to request filenames.  Perhaps look at adding
// some directions to my channel below, just to make it obvious what is going
// on, and to provide some type safety.

// This approach is careful, but not full proof.  If something else is writing
// to the target directory, this program could still overwrite it.  However, it
// will step around anything that is already there. It is also aware of any
// files it has blessed for writing.  A foolproof way to make sure we have a
// unique name would be to use the system open with excusive and create flags.
// That approach would probably be portable to MacOS, because it is based on BSD
// UNIX.  I don't think it would be portable to Windoze.

// I also considered using uuids for generating unique names.  It would have
// worked without all the fun of channeling all the threads through the
// goroutine below.  The downside would have been filenames that differed
// significantly from the names on the cards.

// TargetNameGen provides unique names for the target directory.
func TargetNameGen(getTargetQueue chan GetFileNameMsg, targetDir string, transBuff int, debugMode bool) {

	// Need to track what has been given for file names, so we can
	// make sure there are no conflicts.
	targMap := make(map[string]bool)

	for {
		// This thread blocks here until it gets a request on
		// the channel.
		request := <-getTargetQueue

		if debugMode {
			fmt.Printf("Got target name request for: %s\n",
				request.LeafName)
		}

		callbackMsg := new(returnFileNameMsg)

		var tryName string

		// Loop until we have a valid target name
		for i := 0; i < 10000; i++ {

			// Failsafe (I need to give some thought to finding a better way to handle this... :-P)
			if i == 9999 {
				panic("Error renaming:" + request.LeafName)
			}

			if i == 0 {

				tryName = targetDir + "/" + request.LeafName

			} else {
				leafParts := strings.Split(request.LeafName, ".")
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
					targetDir, leafStub, "-", i, leafExt)

				if debugMode {
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

				callbackMsg.WriteLeafName = tryName
				targMap[tryName] = true
				break
			} else {
				// File with the same name is already
				// there.  Check to see if the file is
				// the same as the one I'm trying to
				// write.

				sameStat, err := cardfileutil.IsFileSame(tryName, request.FullName, transBuff)
				if err != nil {
					// May not be best option, but
					// at least I will know
					// something went wrong.
					panic("Failed to get same status.")
				}

				if sameStat {
					callbackMsg.WriteLeafName = tryName
					callbackMsg.SkipFlag = true
					targMap[tryName] = true
					break
				}
			}
		}

		// The send back the result.
		request.Callback <- *callbackMsg
	}
}
