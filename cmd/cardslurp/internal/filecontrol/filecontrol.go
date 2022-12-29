package filecontrol

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
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

// Put the work request and the results in a single structure.
// This makes doing retries easier.
type CardSlurpWork struct {
	ParentDir   string
	LeafName    string
	LeafMode    os.FileMode
	Skipped     bool
	Copied      bool
	RetriesUsed uint64
	MinorErr    []string
	MajorErr    error
}

// LocateFiles - Main spins up a copy of this function for each of the cards we are offloading.
func LocateFiles(fullPath string, doneMsg chan FinishMsg,
	targetNameManager *TargetNameGenManager, workerPool *WorkerPool,
	debugMode bool) {

	rv := new(FinishMsg)
	rv.FullPath = fullPath

	foundFiles := make([]CardSlurpWork, 0)

	err := recurseDir(fullPath, &foundFiles, &debugMode)
	if err != nil {
		// Punch out early
		rv.MajorErr = fmt.Errorf("error recursing path %s: %w", fullPath, err)
		doneMsg <- *rv
		return
	}

	workCh, resultCh := workerPool.GetChannels()

	for _, foundFile := range foundFiles {
		workCh <- foundFile
	}

	// Loop until all the files to be copied have been accounted for.  If we
	// just called cancel next, we would leave some files behind.
	for i := 0; i < len(foundFiles); i++ {

		res := <-resultCh

		if res.Skipped {
			rv.Skipped++
		}

		if res.Copied {
			rv.Copied++
		}

		if len(res.MinorErr) != 0 {
			rv.MinorErrs = append(rv.MinorErrs, res.MinorErr...)
		}

		if res.MajorErr != nil {
			rv.MajorErr = fmt.Errorf("major error copying %s: %w", res.LeafName, res.MajorErr)
			// Major errors kill the copy process early.
			break
		}
	}

	// Let the main routine know we are done.
	doneMsg <- *rv
}

func recurseDir(fullPath string, foundFiles *[]CardSlurpWork, debugMode *bool) error {

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
			foundRec := CardSlurpWork{
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

// TargetNameGenManager - Manage naming of the target filename.  All Calls for
// a new filename require writing to the map, so make this struct compose sync.Mutex
// instead of sync.RWMutex.
type TargetNameGenManager struct {
	sync.Mutex
	knowntargets map[string]bool
	targetDir    string
}

// NewTargetNameGenManager - Constructor for TargetNameGenManager
func NewTargetNameGenManager(targetDir string) *TargetNameGenManager {
	return &TargetNameGenManager{
		Mutex:        sync.Mutex{},
		knowntargets: make(map[string]bool),
		targetDir:    targetDir,
	}
}

func (t *TargetNameGenManager) GetTargetName(fullName string) (string, error) {

	// Lock, to protoect t.knowntargets
	t.Lock()
	defer t.Unlock()

	_, fileName := path.Split(fullName)
	tryName := path.Join([]string{t.targetDir, fileName}...)

	if !t.knowntargets[tryName] {
		t.knowntargets[tryName] = true
		return tryName, nil
	}

	// If we got this far, we have a naming conflict.

	// Since we are going to be appending to the filename, we
	// now need to handle the file extention.
	fileParts := strings.Split(fileName, ".")

	// A bit cheesy, but this should work
	for i := 0; i < 10000; i++ {

		switch {
		case len(fileParts) == 2:
			// Since we have a file extension, work around it for the file name append.
			tryFileNameExt := fmt.Sprintf("%s-%d.%s", fileParts[0], i, fileParts[1])
			tryFullNameExt := path.Join([]string{t.targetDir, tryFileNameExt}...)
			if !t.knowntargets[tryFullNameExt] {
				t.knowntargets[tryFullNameExt] = true
				return tryFullNameExt, nil
			}
		default:
			// Since we got more or fewer parts than expected, just append
			// to the file as is.
			tryFileName := fmt.Sprintf("%s-%d", fileName, i)
			tryFullName := path.Join([]string{t.targetDir, tryFileName}...)
			if !t.knowntargets[tryFullName] {
				t.knowntargets[tryFullName] = true
				return tryFullName, nil
			}
		}
	}

	return "", errors.New("failed to find unique target name")
}

type WorkerPool struct {
	wg         *sync.WaitGroup
	cancel     context.CancelFunc
	workInput  chan CardSlurpWork
	workOutput chan CardSlurpWork
}

func NewWorkerPool(ctx context.Context, poolSize uint64,
	wg *sync.WaitGroup, nameManager *TargetNameGenManager,
	verifications uint64, debugMode bool,
	maxTriesAllowed uint64) *WorkerPool {

	workCtx, cancel := context.WithCancel(ctx)

	// Make the channels buffered, so we hopefully
	// won't block writing to them.
	workInput := make(chan CardSlurpWork, poolSize*poolSize)
	workOutput := make(chan CardSlurpWork, poolSize*poolSize)

	rv := &WorkerPool{
		wg:         wg,
		cancel:     cancel,
		workInput:  workInput,
		workOutput: workOutput,
	}

	for i := 0; i < int(poolSize); i++ {

		wg.Add(1)
		go func(wkCtx context.Context, wg *sync.WaitGroup,
			nameMan *TargetNameGenManager, inWork chan CardSlurpWork,
			outWork chan<- CardSlurpWork, verify uint64,
			debug bool, maxRetries uint64) {

			wg.Done()

		Loop:
			for {
				select {
				case <-wkCtx.Done():
					return
				case wMsg := <-inWork:

					sourceFile := wMsg.ParentDir + "/" + wMsg.LeafName

					targetName, err := nameMan.GetTargetName(sourceFile)
					if err != nil {
						// We failed to get a target name, so don't retry.
						wMsg.MajorErr = fmt.Errorf("error getting target name for %s: %w", sourceFile, err)
						outWork <- wMsg
						continue Loop
					}

					if debug {
						fmt.Printf("Using %s for write name.\n", targetName)
					}

					_, err = cardfileutil.FileCopy(sourceFile, targetName)
					if err != nil {
						// Handle an error copying the file as a major error.
						wMsg.MajorErr = fmt.Errorf(
							"error copying %s to %s: %w", sourceFile, targetName, err)
						outWork <- wMsg
						continue Loop
					}

					sameStat, err := cardfileutil.IsFileSame(sourceFile, targetName, verify)
					if err != nil {
						// Handle an error calling IsFileSame as a major error.
						wMsg.MajorErr = fmt.Errorf(
							"error calling IsFileSame for %s: %w", sourceFile, err)
						outWork <- wMsg
						continue Loop
					}

					if sameStat {
						// Handle a verification error as a minor error.
						fmt.Printf("%s/%s - Done\n", wMsg.ParentDir, wMsg.LeafName)
						wMsg.Copied = true
						outWork <- wMsg
					} else {
						fmt.Printf("File verification did not match for: %s", sourceFile)
						wMsg.MinorErr = append(wMsg.MinorErr,
							fmt.Sprintf("verification failed for %s.", sourceFile))
						if wMsg.RetriesUsed < maxRetries {
							// Send the work request back for another try.
							wMsg.RetriesUsed++
							inWork <- wMsg
						} else {
							wMsg.MajorErr = fmt.Errorf("%s is out of retries", sourceFile)
							outWork <- wMsg
						}
					}
				}
			}

		}(workCtx, wg, nameManager, workInput, workOutput,
			verifications, debugMode, maxTriesAllowed)
	}

	return rv
}

func (w *WorkerPool) GetChannels() (chan CardSlurpWork, chan CardSlurpWork) {
	return w.workInput, w.workOutput
}

func (w *WorkerPool) CloseWorkerPool() {
	w.cancel()
	w.wg.Wait()
}
