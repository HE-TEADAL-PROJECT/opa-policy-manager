package parser

import (
	"flag"
	"os"
	"testing"
)

var (
	updateGolden = flag.Bool("update", false, "Update golden files for tests")
)

func TestMain(m *testing.M) {
	// Parse the upgrade flag before running tests
	flag.Parse()

	// Run the tests
	os.Exit(m.Run())
}
