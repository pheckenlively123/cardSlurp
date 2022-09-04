package filecontrol

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pheckenlively123/cardSlurp/cmd/cardslurp/internal/cardfileutil"
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
	ParentDir string
	LeafName  string
	LeafMode  os.FileMode
}

// LocateFiles - Main spins up a copy of this function for each of the cards we are offloading.
func LocateFiles(fullPath string, doneMsg chan FinishMsg,
	getTargetQueue chan GetFileNameMsg, transBuff int, debugMode bool) {

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

	workChan := make(chan foundFileStr)
	var wg sync.WaitGroup
	var lock sync.Mutex
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fire up some helper go routines to help expedite the copy process.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(lctx context.Context, wg *sync.WaitGroup, work <-chan foundFileStr) {

			defer wg.Done()

			for {
				select {
				case <-lctx.Done():
					return
				case wMsg := <-work:
					sourceFile := wMsg.ParentDir + "/" + wMsg.LeafName

					targFileMsg := GetFileNameMsg{
						Callback: make(chan returnFileNameMsg),
						LeafName: wMsg.LeafName,
						FullName: sourceFile,
					}

					getTargetQueue <- targFileMsg

					callBackMsg := <-targFileMsg.Callback

					if debugMode {
						fmt.Printf("Using %s for write name.\n",
							callBackMsg.WriteLeafName)
					}

					if callBackMsg.SkipFlag {
						lock.Lock()
						rv.Skipped++
						lock.Unlock()
						fmt.Printf("Skipping, because it is already saved in the target: %s\n", wMsg.LeafName)
						continue
					}

					_, err := cardfileutil.NibbleCopy(sourceFile, callBackMsg.WriteLeafName, transBuff)
					if err != nil {
						lock.Lock()
						rv.MinorErrs = append(rv.MinorErrs,
							"Error copying: "+sourceFile)
						lock.Unlock()
						fmt.Print("Error copying: " + sourceFile)
						continue
					}

					sameStat, err := cardfileutil.IsFileSame(sourceFile, callBackMsg.WriteLeafName, transBuff)
					if err != nil {
						lock.Lock()
						rv.MinorErrs = append(rv.MinorErrs,
							"Error checking files are same: "+sourceFile)
						lock.Unlock()
						fmt.Print("Error verifying copy for: " + sourceFile)
						continue
					}

					if sameStat {
						lock.Lock()
						rv.Copied++
						lock.Unlock()
						fmt.Printf("%s/%s - Done\n", wMsg.ParentDir, wMsg.LeafName)
					} else {
						rv.MinorErrs = append(rv.MinorErrs,
							"file verification did not match for: "+sourceFile)
					}
				}
			}
		}(ctx, &wg, workChan)
	}

	for _, foundFile := range foundFiles {
		workChan <- foundFile
	}

	cancel()
	wg.Wait()

	// Let the main routine know we are done.
	doneMsg <- *rv
}

func recurseDir(fullPath string, foundFiles *[]foundFileStr, debugMode *bool) error {

	fmt.Printf("Recursing: %s\n", fullPath)

	dirList, err := os.ReadDir(fullPath)
	if err != nil {
		return err
	}

	for _, dirEntry := range dirList {

		switch {
		case dirEntry.IsDir():
			newPath := fullPath + "/" + dirEntry.Name()
			recErr := recurseDir(newPath, foundFiles, debugMode)
			if recErr != nil {
				return recErr
			}
		default:
			// Just grab everything that is not a directory.  If they gave us a
			// card with special files, we will just get errors trying to copy.
			foundRec := foundFileStr{
				ParentDir: fullPath,
				LeafName:  dirEntry.Name(),
				LeafMode:  dirEntry.Type(),
			}

			*foundFiles = append(*foundFiles, foundRec)

			if *debugMode {
				fmt.Printf("Found: %s/%s\n", fullPath,
					dirEntry.Name())
			}
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

			// Failsafe (I need to give some thought to finding a better way to
			// handle this... :-P)
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
