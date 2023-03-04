package filecontrol

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// WorkerPoolFinishMsg - passed back to main from the per card worker threads.
type WorkerPoolFinishMsg struct {
	Skipped   uint64
	Copied    uint64
	Retries   uint64
	MinorErrs []string
}

type LocateFilesFinishMsg struct {
	ParentDir   string
	FileCount   uint64
	LocateError error
}

// Put the work request and the results in a single structure.
// This makes doing retries easier.
type CardSlurpWork struct {
	parentDir   string
	fileName    string
	targetName  string
	fileTime    time.Time
	skipped     bool
	copied      bool
	retriesUsed uint64
	minorErr    []string
	majorErr    error
}

type CardFileUtilProvider interface {
	IsFileSame(fromFile string, toFile string) (bool, error)
	CardFileCopy(fromFile string, toFile string) (bool, error)
}

func OrchestrateLocate(cardPathList []string, workerPool *WorkerPool,
	debugMode bool) error {

	finishCh := make(chan LocateFilesFinishMsg)

	wg := &sync.WaitGroup{}

	for _, cp := range cardPathList {
		wg.Add(1)
		go locateFiles(cp, wg, workerPool, finishCh, debugMode)
	}

	for i := 0; i < len(cardPathList); i++ {

		locMsg := <-finishCh
		if locMsg.LocateError != nil {
			// If we got a major error, no point in continuing.
			return fmt.Errorf("major error locating files for %s: %w",
				locMsg.ParentDir, locMsg.LocateError)
		}

		fmt.Printf("Located %d files in: %s\n", locMsg.FileCount, locMsg.ParentDir)
	}

	// The LocateFiles goroutines should all have returned by this point,
	// so this is a belt + suspenders measure.
	wg.Wait()

	return nil
}

// locateFiles - Recurse each of the cards for all files.
func locateFiles(fullPath string, wg *sync.WaitGroup,
	workerPool *WorkerPool, locateFinishCh chan LocateFilesFinishMsg,
	debugMode bool) {

	defer wg.Done()

	rv := LocateFilesFinishMsg{
		ParentDir: fullPath,
	}

	foundFiles := make([]CardSlurpWork, 0)

	err := recurseDir(fullPath, &foundFiles, &debugMode)
	if err != nil {
		// Punch out early
		rv.LocateError = fmt.Errorf("error recursing path %s: %w", fullPath, err)
		return
	}

	// Hand off the list of files to the worker pool to copy in parallel.
	for _, foundFile := range foundFiles {
		workerPool.queueFile(foundFile)
		rv.FileCount++
	}

	locateFinishCh <- rv
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
			fileInfo, err := dirEntry.Info()
			if err != nil {
				return fmt.Errorf(
					"error calling Info() for %s: %w", dirEntry.Name(), err)
			}
			foundRec := CardSlurpWork{
				parentDir: fullPath,
				fileName:  dirEntry.Name(),
				fileTime:  fileInfo.ModTime(),
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
	cfu          CardFileUtilProvider
}

// NewTargetNameGenManager - Constructor for TargetNameGenManager
func NewTargetNameGenManager(targetDir string,
	cfu CardFileUtilProvider) (*TargetNameGenManager, error) {

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
		Mutex:        sync.Mutex{},
		knowntargets: make(map[string]bool),
		targetDir:    targetDir,
		cfu:          cfu,
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

func (t *TargetNameGenManager) getTargetName(fullName string, prevTryName string) (string, bool, error) {

	// Lock, to protoect t.knowntargets
	t.Lock()
	defer t.Unlock()

	// Skip the target filename creation log, if we have a known previous name attempt.
	var tryName string
	var fileName string
	if prevTryName == "" {
		_, fileName = path.Split(fullName)
		tryName = path.Join(t.targetDir, fileName)

		if !t.knowntargets[tryName] {
			t.knowntargets[tryName] = true
			return tryName, false, nil
		}
	} else {
		tryName = prevTryName
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
		same, err := t.cfu.IsFileSame(fullName, tryName)
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

	finalTryName := path.Join(t.targetDir, tryFileName)

	if !t.knowntargets[finalTryName] {
		t.knowntargets[finalTryName] = true
		return finalTryName, false, nil
	}

	// Time to give up, and let the caller know we failed.

	return "", false, errors.New("failed to find unique target name")
}

type WorkerPool struct {
	// wg         *sync.WaitGroup
	poolSize   uint64
	queuedWork []CardSlurpWork
	nameOracle *TargetNameGenManager
	debug      bool
	maxRetries uint64
	cfu        CardFileUtilProvider
}

func NewWorkerPool(poolSize uint64, nameManager *TargetNameGenManager,
	debugMode bool, cfu CardFileUtilProvider,
	maxRetries uint64) *WorkerPool {

	rv := &WorkerPool{
		poolSize:   poolSize,
		queuedWork: make([]CardSlurpWork, 0),
		nameOracle: nameManager,
		debug:      debugMode,
		maxRetries: maxRetries,
		cfu:        cfu,
	}

	return rv
}

func (w *WorkerPool) queueFile(workReq CardSlurpWork) {
	w.queuedWork = append(w.queuedWork, workReq)
}

func (w *WorkerPool) ParallelFileCopy() (WorkerPoolFinishMsg, error) {

	// Start by sorting the queued files by modification time.
	// As long as the two camaras time are close, this should cause
	// the cards to offload in parallel.
	sort.Slice(w.queuedWork, func(i, j int) bool {
		return w.queuedWork[i].fileTime.UnixNano() < w.queuedWork[j].fileTime.UnixNano()
	})

	// Make the input and output channels the same size as our work queue,
	// so we don't block writing.

	inputWork := make(chan CardSlurpWork, len(w.queuedWork))
	outputWork := make(chan CardSlurpWork, len(w.queuedWork))

	// Fire up the worker pool to copy in parallel.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}

	for i := 0; i < int(w.poolSize); i++ {

		wg.Add(1)
		go func(wkCtx context.Context, wg *sync.WaitGroup,
			nameMan *TargetNameGenManager, inWork chan CardSlurpWork,
			outWork chan<- CardSlurpWork, debug bool, maxRetries uint64) {

			wg.Done()

		Loop:
			for {
				select {
				case <-wkCtx.Done():
					return
				case wMsg := <-inWork:

					sourceFile := wMsg.parentDir + "/" + wMsg.fileName

					targetName, same, err := nameMan.getTargetName(sourceFile, wMsg.targetName)
					if err != nil {
						// We failed to get a target name, so don't retry.
						wMsg.majorErr = fmt.Errorf("error getting target name for %s: %w", sourceFile, err)
						outWork <- wMsg
						continue Loop
					}

					if same {
						// The naming oracle says this file is already
						// copied, so skip it.
						wMsg.skipped = true
						fmt.Printf("Skipping %s: (already copied...)\n", targetName)
						outWork <- wMsg
						continue Loop
					}

					if debug {
						fmt.Printf("Using %s for write name.\n", targetName)
					}

					_, err = w.cfu.CardFileCopy(sourceFile, targetName)
					if err != nil {
						// Handle an error copying the file as a major error.
						wMsg.majorErr = fmt.Errorf(
							"error copying %s to %s: %w", sourceFile, targetName, err)
						outWork <- wMsg
						continue Loop
					}

					sameStat, err := w.cfu.IsFileSame(sourceFile, targetName)
					if err != nil {
						// Handle an error calling IsFileSame as a major error.
						wMsg.majorErr = fmt.Errorf(
							"error calling IsFileSame for %s: %w", sourceFile, err)
						outWork <- wMsg
						continue Loop
					}

					if sameStat {
						// Handle a verification error as a minor error.
						fmt.Printf("%s/%s - Done\n", wMsg.parentDir, wMsg.fileName)
						wMsg.copied = true
						outWork <- wMsg
					} else {
						fmt.Printf("File verification did not match for: %s\n", sourceFile)
						wMsg.minorErr = append(wMsg.minorErr,
							fmt.Sprintf("verification failed for: %s", sourceFile))
						if wMsg.retriesUsed < maxRetries {
							// Send the work request back for another try.
							wMsg.retriesUsed++
							wMsg.targetName = targetName
							inWork <- wMsg
							fmt.Printf("Requeuing: %s\n", sourceFile)
						} else {
							wMsg.majorErr = fmt.Errorf("%s is out of retries", sourceFile)
							outWork <- wMsg
						}
					}
				}
			}

		}(ctx, wg, w.nameOracle, inputWork, outputWork, w.debug, w.maxRetries)
	}

	// Now that the worker pool is running, feed all the work
	// requests into it.
	for _, wr := range w.queuedWork {
		inputWork <- wr
	}

	rv := WorkerPoolFinishMsg{
		MinorErrs: make([]string, 0),
	}

	// Suck out the results
	for i := 0; i < len(w.queuedWork); i++ {
		res := <-outputWork

		// Handle major errors first.
		if res.majorErr != nil {
			return WorkerPoolFinishMsg{}, fmt.Errorf(
				"major error copying %s: %w", res.fileName, res.majorErr,
			)
		}

		if res.skipped {
			rv.Skipped++
		}

		if res.copied {
			rv.Copied++
		}

		if res.retriesUsed != 0 {
			rv.Retries += res.retriesUsed
		}

		if len(res.minorErr) != 0 {
			rv.MinorErrs = append(rv.MinorErrs, res.minorErr...)
		}
	}

	// Send the worker pool the all done signal, and wait for
	// the worker goroutines to return
	cancel()
	wg.Wait()

	return rv, nil
}
