package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
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

	// Now that we have verified we are working with the same photo shoot, find metadata files.
	glob := fmt.Sprintf("%s/*.%s", opts.source, opts.extension)
	sourceFileList, err := filepath.Glob(glob)
	if err != nil {
		panic("Error globbing source: " + err.Error())
	}

	// Time to make the donuts...move the files...
	for i, cpFile := range sourceFileList {
		err = safeCopy(opts, cpFile)
		if err != nil {
			panic("error copying: " + cpFile + "\n" + err.Error())
		}
		fmt.Printf("Finished %d of %d\n", i+1, len(sourceFileList))
	}
}

func safeCopy(opts *opts, sourceFull string) error {

	_, sourceFile := path.Split(sourceFull)
	targetName := fmt.Sprintf("%s/%s", opts.target, sourceFile)

	if opts.memorex {
		fmt.Printf("Simulating copying: %s to %s\n", sourceFull, targetName)
		return nil
	}

	from, err := os.Open(sourceFull)
	if err != nil {
		return fmt.Errorf("error opening from file: %w", err)
	}
	defer from.Close()

	to, err := os.OpenFile(targetName, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("error opening to file: %w", err)
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return fmt.Errorf("error copying from to to: %w", err)
	}

	fmt.Printf("Copied: %s to %s\n", sourceFull, targetName)

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
	source    string
	target    string
	extension string
	memorex   bool
}

func getopt() (*opts, error) {

	source := flag.String("source", "", "Source directory")
	target := flag.String("target", "", "Target directory")
	extension := flag.String("extension", "xmp", "File extension")
	memorex := flag.Bool("memorex", true, "Is it live, or is it memorex")

	flag.Parse()

	if *source == "" {
		return &opts{}, fmt.Errorf("-source is a required parameter")
	}
	if *target == "" {
		return &opts{}, fmt.Errorf("-target is a required parameter")
	}

	return &opts{
		source:    *source,
		target:    *target,
		extension: *extension,
		memorex:   *memorex,
	}, nil
}
