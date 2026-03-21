package main

import (
	"fmt"
	"os"
)

func printReleaseDiagram(mode, latest, next string, skip, hasGoreleaserDocker, hasCustomConfig bool) {
	fmt.Fprintln(os.Stderr, "")
	if skip {
		fmt.Fprintln(os.Stderr, "⏭️  Skipped — downstream jobs will not run")
		fmt.Fprintln(os.Stderr, "")
		return
	}
	if latest != "" && next != "" {
		fmt.Fprintf(os.Stderr, "  %s → %s\n", latest, next)
	} else if next != "" {
		fmt.Fprintf(os.Stderr, "  Tag: %s\n", next)
	}
	fmt.Fprintln(os.Stderr, "  Jobs:")
	switch mode {
	case ModeRelease, ModeRerelease:
		fmt.Fprintln(os.Stderr, "    goreleaser")
		if hasGoreleaserDocker {
			fmt.Fprintln(os.Stderr, "    docker (via goreleaser)")
		} else {
			fmt.Fprintln(os.Stderr, "    docker (standalone)")
		}
	case ModeDocker:
		fmt.Fprintln(os.Stderr, "    docker")
	case ModeGoreleaser:
		fmt.Fprintln(os.Stderr, "    goreleaser")
	}
	fmt.Fprintln(os.Stderr, "")
}
