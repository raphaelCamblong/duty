// Package names holds the convention's file and directory names. It is the
// one place a duty filename literal may legally appear; every other package
// imports these constants. This package has zero dependencies.
package names

// BoardFile is the index file that marks a directory as a board.
const BoardFile = "BOARD.md"

// ConfigFile is the project config file; its presence marks the tree root.
const ConfigFile = "duty.toml"

// ReadmeFile is the convention doc duty init writes next to the root board.
const ReadmeFile = "README.md"

// ArchiveDir is the name of a board's completed-tasks subdirectory.
const ArchiveDir = "archive"

// TreeDir is the conventional folder a tree lives in below a project root.
const TreeDir = "duty"
