//go:build ignore
// +build ignore

package main

import (
	"os"

	"github.com/magefile/mage/mage"
)

func main() {
	os.Setenv("MAGEFILE_VERBOSE", "1")
	os.Exit(mage.Main())
}
