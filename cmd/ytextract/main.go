// Command ytextract is the development and diagnostic CLI for VoxScripta.
package main

import (
	"flag"
	"fmt"
	"os"
)

var version = "dev"

// main parses command-line arguments and delegates execution to run.
func main() {
	os.Exit(run(os.Args[1:]))
}

// run executes the CLI with args and returns a process exit code.
func run(args []string) int {
	flags := flag.NewFlagSet("ytextract", flag.ContinueOnError)
	showVersion := flags.Bool("version", false, "print version")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Println(version)
		return 0
	}
	fmt.Fprintln(os.Stderr, "transcript acquisition is not implemented yet")
	return 1
}
