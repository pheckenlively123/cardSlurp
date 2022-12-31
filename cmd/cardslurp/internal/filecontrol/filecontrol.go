package filecontrol

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/google/uuid"
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
	goSignal *sync.RWMutex, debugMode bool) {

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

	goSignal.RLock()

	// Hand off the list of files to the worker pool to copy in parallel.
	for _, foundFile := range foundFiles {
		workCh <- foundFile
	}

	// Loop until all the files to be copied have been accounted for.
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
			err := recurseDir(newPath, foundFiles, debugMode)
			if err != nil {
				return fmt.Errorf("error calling recurseDir: %w", err)
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
	knowntargets  map[string]bool
	targetDir     string
	verifications uint64
}

// NewTargetNameGenManager - Constructor for TargetNameGenManager
func NewTargetNameGenManager(targetDir string,
	verifications uint64) (*TargetNameGenManager, error) {

	stat, err := os.Stat(targetDir)
	if err != nil {
		return &TargetNameGenManager{}, fmt.Errorf(
			"error calling stat on targetdir: %w", err)
	}
	if !stat.IsDir() {
		return &TargetNameGenManager{}, fmt.Errorf(
			"%s is not a directory", targetDir)
	}

	rv := &TargetNameGenManager{
		Mutex:         sync.Mutex{},
		knowntargets:  make(map[string]bool),
		targetDir:     targetDir,
		verifications: verifications,
	}

	files, err := os.ReadDir(targetDir)
	if err != nil {
		return &TargetNameGenManager{}, fmt.Errorf(
			"error calling ReadDir: %w", err)
	}

	for _, fl := range files {

		if !fl.Type().IsRegular() {
			return &TargetNameGenManager{}, fmt.Errorf(
				"target directory should only contain normal files: %s",
				fl.Name(),
			)
		}

		knownName := path.Join(targetDir, fl.Name())
		rv.knowntargets[knownName] = true
	}

	return rv, nil
}

func (t *TargetNameGenManager) GetTargetName(fullName string) (string, bool, error) {

	// Lock, to protoect t.knowntargets
	t.Lock()
	defer t.Unlock()

	_, fileName := path.Split(fullName)
	tryName := path.Join(t.targetDir, fileName)

	if strings.HasPrefix(fileName, ".") {
		fmt.Print("break")
	}

	if !t.knowntargets[tryName] {
		t.knowntargets[tryName] = true
		return tryName, false, nil
	}

	// If we got this far, we have a naming conflict.  Start
	// by seeing if the file was already copied successfully.
	// We might be running again, after some sort of failure.

	// Since multiple goroutines may be trying to write the same
	// file name, we can't assume the file has been written yet,
	// so we need to check that it is there, before we compare them.
	_, err := os.Stat(tryName)
	if err == nil {
		// We should be safe to compare the files now.
		same, err := cardfileutil.IsFileSame(fullName, tryName, t.verifications)
		if err != nil {
			return "", false, fmt.Errorf("error calling IsFileSame: %w", err)
		}
		if same {
			// Let the caller know that this file can be skipped, because
			// it was already copied successfully.
			return tryName, true, nil
		}
	}

	// Since we are going to be appending to the filename, we
	// now need to handle the file extention.
	fileParts := strings.Split(fileName, ".")

	// Use an atomic bomb to crack a walnut.  :-P
	uuid, err := uuid.NewUUID()
	if err != nil {
		return "", false, fmt.Errorf("error making uuid: %w", err)
	}
	uuidStr := uuid.String()

	var tryFileName string
	switch len(fileParts) {
	case 1:
		tryFileName = fmt.Sprintf("%s-%s", fileParts[0], uuidStr)
	case 2:
		tryFileName = fmt.Sprintf("%s_%s.%s", fileParts[0], uuidStr, fileParts[1])
	default:
		return "", false, errors.New(
			"unexpected number of periods in fileName: " + fileName)
	}

	if strings.HasPrefix(tryFileName, ".") {
		fmt.Print("break")
	}

	finalTryName := path.Join(t.targetDir, tryFileName)

	if !t.knowntargets[finalTryName] {
		t.knowntargets[finalTryName] = true
		return finalTryName, false, nil
	}

	// Time to give up, and let the caller know we failed.

	return "", false, errors.New("failed to find unique target name")
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
	workInput := make(chan CardSlurpWork, poolSize*poolSize*poolSize)
	workOutput := make(chan CardSlurpWork, poolSize*poolSize*poolSize)

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

					targetName, same, err := nameMan.GetTargetName(sourceFile)
					if err != nil {
						// We failed to get a target name, so don't retry.
						wMsg.MajorErr = fmt.Errorf("error getting target name for %s: %w", sourceFile, err)
						outWork <- wMsg
						continue Loop
					}

					if same {
						// The naming oracle says this file is already
						// copied, so skip it.
						wMsg.Skipped = true
						fmt.Printf("Skipping %s: (already copied...)\n", targetName)
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
						fmt.Printf("File verification did not match for: %s\n", sourceFile)
						wMsg.MinorErr = append(wMsg.MinorErr,
							fmt.Sprintf("verification failed for: %s", sourceFile))
						if wMsg.RetriesUsed < maxRetries {
							// Send the work request back for another try.
							wMsg.RetriesUsed++
							inWork <- wMsg
							fmt.Printf("Retrying: %s\n", sourceFile)
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

func (w *WorkerPool) Close() {
	w.cancel()
	w.wg.Wait()
}
