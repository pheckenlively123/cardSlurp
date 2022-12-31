package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/pheckenlively123/cardSlurp/cmd/cardslurp/internal/filecontrol"
)

func main() {

	// Get command line options.
	opts, err := GetOpts()
	if err != nil {
		// No point in continuing.
		panic("error processing command line arguments: " + err.Error())
	}

	foundCount := 0
	doneQueue := make(chan filecontrol.FinishMsg)

	targetNameManager, err := filecontrol.NewTargetNameGenManager(
		opts.TargetDir, opts.VerifyPasses)
	if err != nil {
		// No point in continuing
		panic("error making target name oracle: " + err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := &sync.WaitGroup{}

	workerPool := filecontrol.NewWorkerPool(ctx, opts.WorkerPool, wg,
		targetNameManager, opts.VerifyPasses, opts.DebugMode,
		opts.MaxRetries)
	defer workerPool.Close()

	// This idea with the RWMutex did not work nearly as well
	// as I thought it would.  I think I'm going to have to gather
	// the list of files from all the cards, and make a master list.
	// Then sort the list based on file ctime or mtime.  Then feed the
	// List into the worker pool.  This is going to require
	// some refactoring.  I can't wait for each LocateFiles
	// to send back it's done message then.  Probably move that
	// to a sync.WaitGroup.  I already have a wg from the workerPool.
	// Probably leverage that one, to tell me when each LocateFiles is done.

	// Normally, the unlock further down would be a signal it is safe
	// to read.  In this case, it tells the goroutine for each card
	// that it can proceed.  This allows us to offload the cards
	// in parallel.  Otherwise, the first card to start locating files
	// fills up the top of the work queue.
	goSignal := &sync.RWMutex{}
	goSignal.Lock()

	for _, mountDir := range opts.MountList {

		// Spawn a thread to offload each card at the
		// same time.
		go filecontrol.LocateFiles(mountDir, doneQueue, targetNameManager,
			workerPool, goSignal, opts.DebugMode)
		foundCount++
	}

	// One second should be enough time for LocateFiles
	// to have recursed the directory.
	time.Sleep(1 * time.Second)
	goSignal.Unlock()

	summary := make([]filecontrol.FinishMsg, 0)

	// Get results from the worker threads.
	for i := 0; i < foundCount; i++ {
		finishResult := <-doneQueue
		if finishResult.MajorErr != nil {
			panic("Major error locating and copying files: " + finishResult.MajorErr.Error())
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

// CmdOpts - All of the options provided from the command line.
type CmdOpts struct {
	TargetDir    string
	MountList    []string
	DebugMode    bool
	WorkerPool   uint64
	MaxRetries   uint64
	VerifyPasses uint64
}

// GetOpts - Return the command line arguments in a CmdOpts struct
func GetOpts() (CmdOpts, error) {

	targetDir := flag.String("targetdir", "", "Target directory for the copied files.")
	mountListStr := flag.String("mountlist", "", "Comma delimited list of mounted cards.")
	debugMode := flag.Bool("debugMode", false, "Print extra debug information.")
	maxRetries := flag.Uint64("maxretries", 5, "Max number of retry attempts.")
	verifyPasses := flag.Uint64("verifypasses", 2, "Number of file verify test passes")
	workerPoolSize := flag.Uint64("workerpool", 15, "Size of the worker pool")

	flag.Parse()

	if *targetDir == "" {
		return CmdOpts{}, errors.New("-targetdir is a required parameter")
	}
	if *mountListStr == "" {
		return CmdOpts{}, errors.New("-mountlist is a required parameter")
	}

	ml := strings.Split(*mountListStr, ",")

	return CmdOpts{
		TargetDir:    *targetDir,
		MountList:    ml,
		DebugMode:    *debugMode,
		MaxRetries:   *maxRetries,
		VerifyPasses: *verifyPasses,
		WorkerPool:   *workerPoolSize,
	}, nil
}
