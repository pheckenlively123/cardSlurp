package main

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/pheckenlively123/cardSlurp/cmd/cardslurp/internal/filecontrol"
)

func main() {

	// Get command line options.
	opts, err := GetOpts()
	if err != nil {
		// No point in continuing.
		panic("error processing command line arguments: " + err.Error())
	}

	nameOracle, err := filecontrol.NewTargetNameGenManager(
		opts.TargetDir, opts.VerifyPasses)
	if err != nil {
		// No point in continuing
		panic("error making target name oracle: " + err.Error())
	}

	workerPool := filecontrol.NewWorkerPool(opts.WorkerPool,
		nameOracle, opts.VerifyPasses, opts.DebugMode,
		opts.MaxRetries)

	err = filecontrol.OrchestrateLocate(opts.MountList, workerPool, opts.DebugMode)
	if err != nil {
		// No point in continuing
		panic("error recursing card directories: " + err.Error())
	}

	finalResults, err := workerPool.ParallelFileCopy()
	if err != nil {
		panic("major error during parallel file copy: " + err.Error())
	}

	fmt.Printf("Skipped: %d - Copied: %d - Retries: %d\n",
		finalResults.Skipped, finalResults.Copied, finalResults.Retries)

	if len(finalResults.MinorErrs) == 0 {
		fmt.Printf("(No errors.)\n")
	} else {
		fmt.Printf("*** ERRORS ***\n")

		for y := range finalResults.MinorErrs {
			fmt.Printf("%s\n", finalResults.MinorErrs[y])
		}
	}
}

// CmdOpts - All of the options provided from the command line.
type CmdOpts struct {
	TargetDir       string
	MountList       []string
	DebugMode       bool
	WorkerPool      uint64
	MaxRetries      uint64
	VerifyPasses    uint64
	VerifyChunkSize uint64
}

// GetOpts - Return the command line arguments in a CmdOpts struct
func GetOpts() (CmdOpts, error) {

	targetDir := flag.String("targetdir", "", "Target directory for the copied files.")
	mountListStr := flag.String("mountlist", "", "Comma delimited list of mounted cards.")
	debugMode := flag.Bool("debugMode", false, "Print extra debug information.")
	maxRetries := flag.Uint64("maxretries", 5, "Max number of retry attempts.")
	verifyPasses := flag.Uint64("verifypasses", 3, "Number of file verify test passes")
	verifyChunkSize := flag.Uint64("verifychunksize", 16384, "Size of the verify chunks")
	workerPoolSize := flag.Uint64("workerpool", 4, "Size of the worker pool")

	flag.Parse()

	if *targetDir == "" {
		return CmdOpts{}, errors.New("-targetdir is a required parameter")
	}

	if *mountListStr == "" {
		return CmdOpts{}, errors.New("-mountlist is a required parameter")
	}

	if *maxRetries == 0 {
		return CmdOpts{}, errors.New("-maxretries must not be zero")
	}

	if *verifyPasses == 0 {
		return CmdOpts{}, errors.New("-verifypasses must not be zero")
	}

	if *verifyChunkSize == 0 {
		return CmdOpts{}, errors.New("-verifychunksize must not be zero")
	}

	ml := strings.Split(*mountListStr, ",")
	if len(ml) == 0 {
		return CmdOpts{}, errors.New("length of -mountlist must not be zero")
	}

	return CmdOpts{
		TargetDir:       *targetDir,
		MountList:       ml,
		DebugMode:       *debugMode,
		MaxRetries:      *maxRetries,
		VerifyPasses:    *verifyPasses,
		VerifyChunkSize: *verifyChunkSize,
		WorkerPool:      *workerPoolSize,
	}, nil
}
