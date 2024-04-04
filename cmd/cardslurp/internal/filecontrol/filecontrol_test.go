package filecontrol

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/pheckenlively123/cardSlurp/internal/cardfileutil"
)

/*
func TestRecurseDir(t *testing.T) {

	// Base the path of the test source on the home directory variable.  That
	// way, this test should work with other developers too.  :-P
	homedir := os.Getenv("HOME")

	foundFiles := make([]CardSlurpWork, 0)
	fullPath := homedir + "/go/src/github.com/cardSlurp/cmd/cardslurp/internal/filecontrol/testData/source"
	debugMode := false
	err := recurseDir(fullPath, &foundFiles, &debugMode)
	if err != nil {
		t.Fatal("recurseDir() returned an error")
	}
}
*/

var (
	errInjected = errors.New("injected error")
)

type CardFileUtilMock struct {
	cfu          cardfileutil.CardFileUtil
	perturbation rand.Rand
}

func NewCardFileUtilMock() *CardFileUtilMock {
	source := rand.NewSource(time.Now().UnixMicro())
	return &CardFileUtilMock{
		cfu:          *cardfileutil.NewCardFileUtil(16384, 3),
		perturbation: *rand.New(source),
	}
}

func (c *CardFileUtilMock) IsFileSame(fromFile string, toFile string) (bool, error) {
	// 10% of the time, throw and a simulated major error.  10% of the time have
	// verification fail, but no error.  The rest of the time, return the results
	// of the actual verify method.
	dice := c.perturbation.Int63n(10)
	switch dice {
	case 8:
		return false, nil
	case 9:
		return false, errInjected
	default:
		return c.cfu.IsFileSame(fromFile, toFile)
	}
}

func (c *CardFileUtilMock) CardFileCopy(fromFile string, toFile string) error {
	// 10% of the time, throw and error instead of calling the corresponding cfu method.
	dice := c.perturbation.Int63n(10)
	if dice == 9 {
		return errInjected
	}
	return c.cfu.CardFileCopy(fromFile, toFile)
}

// This test mimics the behavior of the main application.
// The main purpose is to exersize everything in dlv.
func TestNewWorkerPool(t *testing.T) {

	homedir := os.Getenv("HOME")

	testDir := homedir + "/go/src/github.com/pheckenlively123/cardSlurp/cmd/cardslurp/internal/filecontrol/testData"

	cardA := testDir + "/source/A"
	cardB := testDir + "/source/B"
	cardC := testDir + "/source/C"
	cardD := testDir + "/source/D"

	targetDir := testDir + "/target"

	// Try to remove the targetdir, in case a test died in the middle.
	// Then make it.
	_ = os.RemoveAll(targetDir)
	err := os.MkdirAll(targetDir, 0777)
	if err != nil {
		t.Fatal("error making targetdir: " + err.Error())
	}

	cfum := NewCardFileUtilMock()

	nameOracle, err := NewTargetNameGenManager(targetDir, cfum)
	if err != nil {
		t.Fatal("error making name oracle: " + err.Error())
	}

	workerPool := NewWorkerPool(4, nameOracle, false, cfum, 5)

	err = OrchestrateLocate([]string{cardA, cardB, cardC, cardD},
		workerPool, true)
	if err != nil && !errors.Is(err, errInjected) {
		t.Fatal("unexpected from OrchestrateLocate: " + err.Error())
	}

	finalResults, err := workerPool.ParallelFileCopy()
	if err != nil && !errors.Is(err, errInjected) {
		t.Fatal("unexpected error from parallel file copy: " + err.Error())
	}

	// Print the summary results.
	fmt.Printf("Skipped: %d - Copied: %d\n", finalResults.Skipped,
		finalResults.Copied)
	if len(finalResults.MinorErrs) == 0 {
		fmt.Printf("(No errors.)\n")
	} else {
		fmt.Printf("*** ERRORS ***\n")

		for y := range finalResults.MinorErrs {
			fmt.Printf("%s\n", finalResults.MinorErrs[y])
		}
	}
}
