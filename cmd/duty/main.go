// Command duty is a file-based task system: markdown task files plus nested
// board indexes, driven by one binary. Real dispatch lands in a later task;
// for now it only prints usage.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "usage: duty <command> [args]")
	os.Exit(2)
}
