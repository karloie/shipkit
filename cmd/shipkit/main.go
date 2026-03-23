package main

import (
	"fmt"
	"os"
	"strings"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

func main() {
	// Handle global flags
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "-v", "--version":
			printVersion()
			return
		case "-h", "--help":
			printHelp()
			return
		}
	}

	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "plan":
		err = runPlan(os.Args[2:])
	case "build":
		err = runBuild(os.Args[2:])
	case "publish":
		err = runPublish(os.Args[2:])
	case "publish-goreleaser":
		err = runPublishGoreleaser(os.Args[2:])
	case "publish-docker":
		err = runPublishDocker(os.Args[2:])
	case "decide":
		err = runDecide(os.Args[2:])
	case "summary":
		err = runSummary(os.Args[2:])
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
		// Deprecated: use publish-goreleaser
		err = runGoReleaser(os.Args[2:])
	case "verify-version":
		err = runVerifyVersion(os.Args[2:])
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

func logInputs(inputs map[string]string) {
	if len(inputs) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "INPUT:")
	for key, value := range inputs {
		fmt.Fprintf(os.Stderr, " - %s=%s\n", key, value)
	}
	fmt.Fprintln(os.Stderr, "")
}

func logOutputs(outputs map[string]string) {
	if len(outputs) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "OUTPUT:")
	for key, value := range outputs {
		fmt.Fprintf(os.Stderr, " - %s=%s\n", key, value)
	}
	fmt.Fprintln(os.Stderr, "")
}

func printVersion() {
	fmt.Printf("shipkit %s\n", buildVersion)
	fmt.Printf("  commit: %s\n", buildCommit)
	fmt.Printf("  built:  %s\n", buildDate)
}

func printHelp() {
	fmt.Fprintln(os.Stderr, "shipkit - Release automation toolkit")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Usage: shipkit <subcommand> [options]")
	fmt.Fprintln(os.Stderr, "       shipkit -v, --version")
	fmt.Fprintln(os.Stderr, "       shipkit -h, --help")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  version               Compute next release version")
	fmt.Fprintln(os.Stderr, "  plan                  Plan release workflow")
	fmt.Fprintln(os.Stderr, "  build                 Execute build using Make/just/task")
	fmt.Fprintln(os.Stderr, "  decide                Validate build results and decide on publishing")
	fmt.Fprintln(os.Stderr, "  publish               Execute publish using Make/just/task")
	fmt.Fprintln(os.Stderr, "  publish-goreleaser    Publish release using GoReleaser")
	fmt.Fprintln(os.Stderr, "  publish-docker        Build and publish Docker images")
	fmt.Fprintln(os.Stderr, "  summary               Generate release summary")
	fmt.Fprintln(os.Stderr, "  goreleaser            Generate GoReleaser config")
	fmt.Fprintln(os.Stderr, "  docker-hub-readme     Update Docker Hub README")
	fmt.Fprintln(os.Stderr, "  docker-hub-status     Check Docker Hub repository status")
	fmt.Fprintln(os.Stderr, "  git-config            Configure git for release")
	fmt.Fprintln(os.Stderr, "  git-tag               Create and push git tag")
	fmt.Fprintln(os.Stderr, "  git-tag-cleanup       Delete tag on release failure")
	fmt.Fprintln(os.Stderr, "  assets-delete         Delete release assets")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Options:")
	fmt.Fprintln(os.Stderr, "  -v, --version         Show version information")
	fmt.Fprintln(os.Stderr, "  -h, --help            Show this help message")
}
