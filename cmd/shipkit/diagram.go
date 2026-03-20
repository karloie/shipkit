package main

import (
	"fmt"
	"os"
)

func printReleaseDiagram(mode, latest, next string, dryRun, hasGoreleaserDocker, hasCustomConfig bool) {
	fmt.Fprintf(os.Stderr, "\n🔄 Version: %s → %s\n", latest, next)
	if dryRun {
		fmt.Fprintln(os.Stderr, "⏭️  Status: Skipping (no release markers)")
	} else {
		fmt.Fprintf(os.Stderr, "✓ Mode: %s\n", mode)
		if mode == ModeRelease || mode == ModeRerelease {
			if hasGoreleaserDocker {
				fmt.Fprintln(os.Stderr, "🐳 Docker: Included in GoReleaser")
			} else {
				fmt.Fprintln(os.Stderr, "🐳 Docker: Standalone (runs in parallel)")
			}
			if hasCustomConfig {
				fmt.Fprintln(os.Stderr, "📝 Config: Custom .goreleaser.yml")
			} else {
				fmt.Fprintln(os.Stderr, "🤖 Config: Auto-generated")
			}
		}
	}
	fmt.Fprintln(os.Stderr, "")
}
