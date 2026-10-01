package main

import "fmt"

// These public build fields are populated through linker flags.
var version = "dev"
var commit = "unknown"
var buildDate = "unknown"

func versionInfo() string {
	return fmt.Sprintf("DevDock %s (commit %s, built %s)", version, commit, buildDate)
}
