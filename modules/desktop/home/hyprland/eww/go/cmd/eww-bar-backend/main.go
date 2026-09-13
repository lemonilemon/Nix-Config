// Command eww-bar-backend is the eww bar's state daemon: one JSON snapshot per
// line on stdout for eww's deflisten, plus the eww-barctl control socket.
package main

import (
	"fmt"
	"os"

	"ewwbar/internal/app"
)

func main() {
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
