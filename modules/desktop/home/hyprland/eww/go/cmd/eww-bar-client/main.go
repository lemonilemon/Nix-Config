// Command eww-bar-client backs eww-barctl and eww-popup.
//
// eww.yuck invokes them from 39 handlers including :onscroll, and they do almost
// no work: build a payload, round-trip a unix socket, or shell out to eww. The
// daemon answers a ping in 0.06 ms, so essentially all of the click latency was
// CPython startup -- 45 ms for eww-barctl and 51 ms for eww-popup on this
// laptop, against ~3 ms here.
//
// That is why the clients were ported first. The daemon followed for a different
// reason: 93% of its CPU is the child processes its collectors fork, which cost
// the same in any language, so the win there was not speed but not having to
// maintain two implementations of the same collectors.
package main

import (
	"os"
	"path/filepath"

	"ewwbar/internal/ipc"
	"ewwbar/internal/popup"
)

func main() {
	switch filepath.Base(os.Args[0]) {
	case "eww-barctl":
		os.Exit(ipc.Run(os.Args[1:]))
	case "eww-popup":
		os.Exit(popup.Run(os.Args[1:]))
	}

	// Invoked under any other name (the multi-call binary itself, or a test
	// harness): take the mode from argv[1].
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "ctl":
			os.Exit(ipc.Run(os.Args[2:]))
		case "popup":
			os.Exit(popup.Run(os.Args[2:]))
		}
	}
	os.Stderr.WriteString("usage: eww-bar-client ctl <args> | popup <args>\n")
	os.Exit(1)
}
