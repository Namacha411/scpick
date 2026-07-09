// Command scpick is an interactive SCP/SFTP file transfer tool. See
// SPEC.md for the full design.
package main

import (
	"fmt"
	"os"

	"scpick/internal/transfer"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: scpick <pull|push>")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "pull":
		err = transfer.RunPull()
	case "push":
		err = transfer.RunPush()
	default:
		fmt.Fprintf(os.Stderr, "scpick: unknown command %q (want pull or push)\n", os.Args[1])
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "scpick: %v\n", err)
		os.Exit(1)
	}
}
