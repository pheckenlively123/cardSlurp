package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pheckenlively123/cardSlurp/internal/cardfileutil"
)

func main() {

	opts, err := getopt()
	if err != nil {
		panic("Error processing command line arguments: " + err.Error())
	}

	// Before we work on copying things, let's make sure the source and target are the same photo shoot.
	if !checkLastDir(opts.source, opts.target) {
		panic("Source and target appear to be different photo shoots")
	}

	// Make a backup directory in the target directory, so we
	// can backup the side cart files.  However, first make sure
	// the we are not stepping on any names already present in the
	// directory.
	backupDir := fmt.Sprintf("%s/sideCartBackup-%d", opts.target, time.Now().UnixMilli())
	// We want the call below to return an error, because that will mean the backup directory name does not exist yet.
	_, err = os.Lstat(backupDir)
	if !os.IsNotExist(err) {
		panic("Got unexpected error from stat call: " + err.Error())
	}
	err = os.Mkdir(backupDir, fs.ModeDir|0777)
	if err != nil {
		panic("Error making backup directory: " + err.Error())
	}

	// Before copying side cart files from the source, back up the side cart
	// files that are already in the target location.
	targetGlobString := fmt.Sprintf("%s/*.%s", opts.target, opts.extension)
	targetFileList, err := filepath.Glob(targetGlobString)
	if err != nil {
		panic("error globing target: " + err.Error())
	}
	fmt.Printf("Backing up %s files in %s to %s...\n", opts.extension, opts.target, backupDir)
	for i, targetFileFullPath := range targetFileList {
		_, backupFileName := path.Split(targetFileFullPath)
		backupName := fmt.Sprintf("%s/%s", backupDir, backupFileName)
		err = os.Rename(targetFileFullPath, backupName)
		if err != nil {
			panic("Error backing up " + targetFileFullPath + " : " + err.Error())
		}
		fmt.Printf("Saved %s to backup: (%d of %d)\n", targetFileFullPath, i+1, len(targetFileList))
	}

	// Now that we have verified we are working with the same
	// photo shoot, find metadata files.
	sourceGlobString := fmt.Sprintf("%s/*.%s", opts.source, opts.extension)
	sourceFileList, err := filepath.Glob(sourceGlobString)
	if err != nil {
		panic("Error globbing source: " + err.Error())
	}

	cfu := cardfileutil.NewCardFileUtil(opts.verifyChunkSize, opts.verifyPasses)

	// Time to make the donuts...move the files...
	for i, cpFile := range sourceFileList {
		err = safeCopy(cfu, opts, cpFile)
		if err != nil {
			panic("error copying: " + cpFile + "\n" + err.Error())
		}

		fmt.Printf("Finished %s (%d of %d)\n", cpFile, i+1, len(sourceFileList))
	}
}

func safeCopy(cfu *cardfileutil.CardFileUtil, opts *opts, fullSourcePath string) error {

	_, sourceFile := path.Split(fullSourcePath)
	targetName := fmt.Sprintf("%s/%s", opts.target, sourceFile)

	if fullSourcePath == targetName {
		return errors.New("source and target are the same")
	}

	if opts.memorex {
		fmt.Printf("Simulating copying: %s to %s\n", fullSourcePath, targetName)
		return nil
	}

	err := cfu.CardFileCopy(fullSourcePath, targetName)
	if err != nil {
		return fmt.Errorf("error calling CardFileCopy: %w", err)
	}

	fileOK, err := cfu.IsFileSame(fullSourcePath, targetName)
	if err != nil {
		return fmt.Errorf("error calling IsFileSame: %w", err)
	}
	if !fileOK {
		return fmt.Errorf("error verifying %s copied OK", fullSourcePath)
	}

	return nil
}

func checkLastDir(source string, target string) bool {

	_, sourcePath := path.Split(source)
	_, targetPath := path.Split(target)

	if sourcePath == targetPath {
		return true
	} else {
		return false
	}
}

type opts struct {
	source          string
	target          string
	extension       string
	memorex         bool
	verifyPasses    uint64
	verifyChunkSize uint64
}

func getopt() (*opts, error) {

	source := flag.String("source", "", "Source directory")
	target := flag.String("target", "", "Target directory")
	extension := flag.String("extension", "xmp", "File extension")
	verifyPasses := flag.Uint64("verifypasses", 3, "Number of file verify test passes")
	verifyChunkSize := flag.Uint64("verifychunksize", 16384, "Size of the verify chunks")
	memorex := flag.Bool("memorex", true, "Is it live, or is it memorex")

	flag.Parse()

	if *source == "" {
		return &opts{}, errors.New("-source is a required parameter")
	}
	if *target == "" {
		return &opts{}, errors.New("-target is a required parameter")
	}
	if *source == *target {
		return &opts{}, errors.New("-source and -target must not be the same")
	}

	return &opts{
		source:          *source,
		target:          *target,
		extension:       *extension,
		memorex:         *memorex,
		verifyPasses:    *verifyPasses,
		verifyChunkSize: *verifyChunkSize,
	}, nil
}
