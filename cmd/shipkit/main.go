package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: shipkit <subcommand> [options]")
		fmt.Fprintln(os.Stderr, "Subcommands: version, plan, assets-delete, goreleaser, docker-hub-readme, git-config, git-tag, git-tag-cleanup, docker-hub-status")
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "plan":
		err = runPlan(os.Args[2:])
	case "assets-delete":
		err = runAssetsDelete(os.Args[2:])
	case "docker-hub-status":
		err = runDockerHubStatus(os.Args[2:])
	case "docker-hub-readme":
		err = runDockerHubReadme(os.Args[2:])
	case "git-config":
		err = runGitConfig(os.Args[2:])
	case "git-tag":
		err = runGitTag(os.Args[2:])
	case "git-tag-cleanup":
		err = runGitTagCleanup(os.Args[2:])
	case "goreleaser":
		err = runGoReleaser(os.Args[2:])
	case "version":
		err = runVersion(os.Args[2:])
	default:
		err = fmt.Errorf("unknown subcommand: %s", os.Args[1])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func writeOutput(outputFile, key, value string) {
	if outputFile == "" {
		fmt.Printf("%s=%s\n", key, value)
		return
	}
	f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: could not write to GITHUB_OUTPUT: %v\n", err)
		return
	}
	defer f.Close()
	if strings.Contains(value, "\n") {
		delim := "EOF"
		fmt.Fprintf(f, "%s<<%s\n%s\n%s\n", key, delim, value, delim)
		return
	}
	fmt.Fprintf(f, "%s=%s\n", key, value)
}
