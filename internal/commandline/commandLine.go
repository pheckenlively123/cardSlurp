package commandline

import (
	"errors"
	"flag"
	"strings"
)

// CmdOpts - All of the options provided from the command line.
type CmdOpts struct {
	TargetDir string
	MountList []string
	DebugMode bool
	TransBuff int
}

// GetOpts - Return the command line arguments in a CmdOpts struct
func GetOpts() (*CmdOpts, error) {

	targetDir := flag.String("targetdir", "", "Target directory for the copied files.")
	mountList := flag.String("mountlist", "", "Comma delimited list of card mount points.")
	debugMode := flag.Bool("debugmode", false, "Print extra debug information.")
	transBuff := flag.Int("transbuff", 8192, "Transfer buffer size.")

	flag.Parse()

	if *targetDir == "" {
		return &CmdOpts{}, errors.New("-targetdir is a required parameter")
	}

	if *mountList == "" {
		return &CmdOpts{}, errors.New("-mountlist is a required parameter")
	}

	rv := &CmdOpts{
		TargetDir: *targetDir,
		MountList: strings.Split(*mountList, ","),
		DebugMode: *debugMode,
		TransBuff: *transBuff,
	}

	return rv, nil
}
