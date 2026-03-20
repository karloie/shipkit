package main

import (
	"fmt"
	"os"
)

func printReleaseDiagram(mode, latest, next string, dryRun, hasGoreleaserDocker, hasCustomConfig bool) {
	// Write to GitHub Step Summary for visual display in Actions UI
	githubSummary := os.Getenv("GITHUB_STEP_SUMMARY")

	if githubSummary != "" {
		writeGitHubStepSummary(githubSummary, mode, latest, next, dryRun, hasGoreleaserDocker, hasCustomConfig)
	}

	// Simple console output
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

func writeGitHubStepSummary(summaryFile, mode, latest, next string, dryRun, hasGoreleaserDocker, hasCustomConfig bool) {
	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	if dryRun {
		fmt.Fprintf(f, "## ⏭️ Release Skipped\n\n")
		fmt.Fprintf(f, "No release markers found. Skipping tag creation and publish.\n\n")
		return
	}

	fmt.Fprintf(f, "## 🚀 Release Execution Plan\n\n")
	fmt.Fprintf(f, "**Version:** `%s` → `%s`\n\n", latest, next)

	switch mode {
	case ModeRelease, ModeRerelease:
		if hasGoreleaserDocker {
			fmt.Fprintf(f, "```mermaid\n")
			fmt.Fprintf(f, "graph LR\n")
			fmt.Fprintf(f, "    Plan[\"📋 Plan\"] --> GoReleaser[\"📦 GoReleaser\"]\n")
			fmt.Fprintf(f, "    GoReleaser --> Docker[\"🐳 Docker\"]\n")
			fmt.Fprintf(f, "    style Plan fill:#e1f5ff,stroke:#0366d6,stroke-width:2px\n")
			fmt.Fprintf(f, "    style GoReleaser fill:#fff3cd,stroke:#ffc107,stroke-width:2px\n")
			fmt.Fprintf(f, "    style Docker fill:#d4edda,stroke:#28a745,stroke-width:2px\n")
			fmt.Fprintf(f, "```\n\n")
			fmt.Fprintf(f, "- 🐳 **Docker:** Handled by GoReleaser\n")
		} else {
			fmt.Fprintf(f, "```mermaid\n")
			fmt.Fprintf(f, "graph LR\n")
			fmt.Fprintf(f, "    Plan[\"📋 Plan\"] --> Docker[\"🐳 Docker<br/>⚡ FAST\"]\n")
			fmt.Fprintf(f, "    Plan --> GoReleaser[\"📦 GoReleaser<br/>(slower)\"]\n")
			fmt.Fprintf(f, "    style Plan fill:#e1f5ff,stroke:#0366d6,stroke-width:2px\n")
			fmt.Fprintf(f, "    style Docker fill:#d4edda,stroke:#28a745,stroke-width:3px\n")
			fmt.Fprintf(f, "    style GoReleaser fill:#fff3cd,stroke:#ffc107,stroke-width:2px\n")
			fmt.Fprintf(f, "```\n\n")
			fmt.Fprintf(f, "- 🐳 **Docker:** Standalone - publishes immediately in parallel\n")
		}
		if hasCustomConfig {
			fmt.Fprintf(f, "- 📝 **GoReleaser:** Custom `.goreleaser.yml` config\n")
		} else {
			fmt.Fprintf(f, "- 🤖 **GoReleaser:** Auto-generated config\n")
		}
	case ModeDocker:
		fmt.Fprintf(f, "```mermaid\n")
		fmt.Fprintf(f, "graph LR\n")
		fmt.Fprintf(f, "    Docker[\"🐳 Docker Only<br/>⚡ Fast publish\"]\n")
		fmt.Fprintf(f, "    style Docker fill:#d4edda,stroke:#28a745,stroke-width:3px\n")
		fmt.Fprintf(f, "```\n\n")
		fmt.Fprintf(f, "- ⚡ **Mode:** Docker-only publish (no GoReleaser)\n")
	case ModeGoreleaser:
		fmt.Fprintf(f, "```mermaid\n")
		fmt.Fprintf(f, "graph LR\n")
		fmt.Fprintf(f, "    GoReleaser[\"📦 GoReleaser<br/>Binaries only\"]\n")
		fmt.Fprintf(f, "    style GoReleaser fill:#fff3cd,stroke:#ffc107,stroke-width:2px\n")
		fmt.Fprintf(f, "```\n\n")
		if hasCustomConfig {
			fmt.Fprintf(f, "- 📝 **Config:** Custom `.goreleaser.yml`\n")
		} else {
			fmt.Fprintf(f, "- 🤖 **Config:** Auto-generated\n")
		}
		fmt.Fprintf(f, "- 🚫 **Docker:** Disabled\n")
	}

	fmt.Fprintf(f, "\n")
}
