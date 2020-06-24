package commandline

import "flag"

// CmdOpts - All of the options provided from the command line.
type CmdOpts struct {
	TargetDir string
	MountDir  string
	SearchStr string
	DebugMode bool
	TransBuff int
}

var targetDir = flag.String("targetDir", "", "Target directory for the copied files.")
var mountDir = flag.String("mountDir", "", "Directory where cards are mounted.")
var searchStr = flag.String("searchStr", "", "String to distinguish cards from other mounted media in mountDir.")
var debugMode = flag.Bool("debugMode", false, "Print extra debug information.")
var transBuff = flag.Int("transBuff", 8192, "Transfer buffer size.")

func init() {
	flag.Parse()

	if *targetDir == "" {
		flag.PrintDefaults()
		panic("Missing -targetDir\n")
	}

	if *mountDir == "" {
		flag.PrintDefaults()
		panic("Missing -mountDir\n")
	}

	if *searchStr == "" {
		flag.PrintDefaults()
		panic("Missing -searchStr\n")
	}
}

// GetOpts - Return the command line arguments in a CmdOpts struct
func GetOpts() CmdOpts {

	rv := new(CmdOpts)

	rv.TargetDir = *targetDir
	rv.MountDir = *mountDir
	rv.SearchStr = *searchStr
	rv.DebugMode = *debugMode
	rv.TransBuff = *transBuff

	return *rv
}
