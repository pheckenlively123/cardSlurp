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
func GetOpts() (CmdOpts, error) {

	targetDir := flag.String("targetdir", "", "Target directory for the copied files.")
	mountListStr := flag.String("mountlist", "", "Directory where cards are mounted.")
	debugMode := flag.Bool("debugMode", false, "Print extra debug information.")
	transBuff := flag.Int("transBuff", 8192, "Transfer buffer size.")

	flag.Parse()

	if *targetDir == "" {
		return CmdOpts{}, errors.New("-targetdir is a required parameter")
	}
	if *mountListStr == "" {
		return CmdOpts{}, errors.New("-mountlist is a required parameter")
	}

	ml := strings.Split(*mountListStr, ",")

	return CmdOpts{
		TargetDir: *targetDir,
		MountList: ml,
		DebugMode: *debugMode,
		TransBuff: *transBuff,
	}, nil
}
