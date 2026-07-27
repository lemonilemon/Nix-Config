// Command eww-bar-client is the compiled half of the eww bar backend.
//
// It serves eww-barctl and eww-popup, which eww.yuck invokes from 39 handlers
// including :onscroll. Those are pure client work -- build a payload, round-trip
// a unix socket, or shell out to eww -- and the daemon answers a ping in 0.06 ms,
// so essentially all of the click latency was CPython startup: 45 ms for
// eww-barctl and 51 ms for eww-popup on this laptop, against ~3 ms here.
//
// The daemon itself stays in Python. Measured, 93% of its CPU is the child
// processes its collectors fork, and those cost the same in any language.
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
