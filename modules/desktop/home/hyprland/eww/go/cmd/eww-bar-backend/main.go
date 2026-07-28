// Command eww-bar-backend is the eww bar's state daemon.
//
// It writes one JSON snapshot per line to stdout, which eww's `deflisten`
// reads, and serves the control socket eww-barctl talks to.
package main

import (
	"fmt"
	"os"

	"ewwbar/internal/app"
)

func main() {
	// argv[1] is accepted and checked rather than ignored: the eww.yuck
	// deflisten invokes `eww-bar-backend bar`, and silently accepting any word
	// there would hide a typo in the widget definition as a working daemon.
	mode := "bar"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	if mode != "bar" {
		fmt.Fprintf(os.Stderr, "unknown backend mode: %s\n", mode)
		os.Exit(1)
	}
	os.Exit(app.Run())
}
