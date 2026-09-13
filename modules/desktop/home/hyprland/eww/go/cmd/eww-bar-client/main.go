// Command eww-bar-client backs eww-barctl and eww-popup.
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
