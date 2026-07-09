// Command duty is a file-based task system: markdown task files plus nested
// board indexes, kept in sync by one binary. main is a thin delegate — all
// dispatch, flag parsing, and error rendering live in internal/cli.
package main

import (
	"os"

	"github.com/raphaelCamblong/duty/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
