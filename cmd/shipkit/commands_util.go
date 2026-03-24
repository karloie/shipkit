package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runEnv exports plan.json fields as environment variables
func runEnv(args []string) error {
	// Load plan.json
	plan, err := loadPlan("plan.json")
	if err != nil {
		return fmt.Errorf("failed to load plan.json: %w", err)
	}

	// Get version
	version := plan.TagRelease
	if version == "" {
		version = plan.TagNext
	}
	if version == "" {
		version = "dev"
	}

	// Get commit
	commit := plan.Sha
	if commit == "" {
		commit = getCurrentCommit()
	}

	// Get date
	date := getCurrentDate()

	// Output export statements
	fmt.Printf("export BUILD_VERSION=%s\n", version)
	fmt.Printf("export BUILD_COMMIT=%s\n", commit)
	fmt.Printf("export BUILD_DATE=%s\n", date)
	fmt.Printf("export DOCKER_IMAGE=%s\n", plan.DockerImage)
	fmt.Printf("export DRY_RUN=%t\n", plan.DryRun)
	fmt.Printf("export BUILD_MODE=%s\n", plan.Mode)

	return nil
}

// runGoBuild builds a Go binary with version information from plan.json
func runGoBuild(args []string) error {
	fs := newFlagSet("go-build")
	output := fs.String("output", "", "Output binary name (required)")
	mainPkg := fs.String("main", "./cmd/...", "Main package path")
	ldflags := fs.String("ldflags", "", "Additional ldflags")

	parseFlagsOrExit(fs, args)

	if *output == "" {
		return fmt.Errorf("--output is required")
	}

	// Load plan.json
	plan, err := loadPlan("plan.json")
	if err != nil {
		return fmt.Errorf("failed to load plan.json: %w", err)
	}

	// Get version info
	version := plan.TagRelease
	if version == "" {
		version = plan.TagNext
	}
	if version == "" {
		version = "dev"
	}

	commit := plan.Sha
	if commit == "" {
		commit = getCurrentCommit()
	}

	date := getCurrentDate()

	logInputs(map[string]string{
		"output":  *output,
		"main":    *mainPkg,
		"version": version,
		"commit":  commit,
		"date":    date,
	})

	// Build ldflags with version info
	versionLdflags := fmt.Sprintf("-X main.Version=%s -X main.GitCommit=%s -X main.BuildTime=%s", version, commit, date)
	if *ldflags != "" {
		versionLdflags = versionLdflags + " " + *ldflags
	}

	// Run go build
	cmdArgs := []string{"build", "-ldflags", versionLdflags, "-o", *output, *mainPkg}
	logCommand("go", cmdArgs...)

	cmd := exec.Command("go", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logError(fmt.Sprintf("go build failed: %v", err))
		return err
	}

	// Get output size
	info, err := os.Stat(*output)
	if err == nil {
		size := float64(info.Size()) / 1024 / 1024
		logSuccess(fmt.Sprintf("Built %s (%.2f MB)", *output, size))
	} else {
		logSuccess(fmt.Sprintf("Built %s", *output))
	}

	logOutputs(map[string]string{
		"binary":  *output,
		"version": version,
	})

	return nil
}

// runDockerUtil handles Docker build and push operations
func runDockerUtil(args []string) error {
	fs := newFlagSet("docker")
	release := fs.Bool("release", false, "Build and push to registry")
	image := fs.String("image", "", "Override image name from plan.json")
	file := fs.String("file", "Containerfile", "Dockerfile to use")
	platform := fs.String("platform", "", "Platforms (e.g., linux/amd64,linux/arm64)")
	tagsOnly := fs.Bool("tags-only", false, "Only output tag list")

	parseFlagsOrExit(fs, args)

	// Load plan.json
	plan, err := loadPlan("plan.json")
	if err != nil {
		return fmt.Errorf("failed to load plan.json: %w", err)
	}

	// Get version info
	version := plan.TagRelease
	if version == "" {
		version = plan.TagNext
	}
	if version == "" {
		version = "dev"
	}

	commit := plan.Sha
	if commit == "" {
		commit = getCurrentCommit()
	}

	date := getCurrentDate()

	// Get image name
	imageName := *image
	if imageName == "" {
		imageName = plan.DockerImage
	}
	if imageName == "" {
		return fmt.Errorf("no image specified (use --image or set docker_image in plan.json)")
	}

	// Build tag list
	tags := []string{
		fmt.Sprintf("%s:%s", imageName, version),
		fmt.Sprintf("%s:latest", imageName),
	}

	// If --tags-only, just output tags and exit
	if *tagsOnly {
		for _, tag := range tags {
			fmt.Println(tag)
		}
		return nil
	}

	logInputs(map[string]string{
		"image":    imageName,
		"version":  version,
		"commit":   commit,
		"date":     date,
		"file":     *file,
		"platform": *platform,
		"release":  fmt.Sprintf("%t", *release),
		"dry_run":  fmt.Sprintf("%t", plan.DryRun),
	})

	// Build docker command
	dockerCmd := []string{"build"}

	// Add build args
	dockerCmd = append(dockerCmd,
		"--build-arg", fmt.Sprintf("BUILD_VERSION=%s", version),
		"--build-arg", fmt.Sprintf("BUILD_COMMIT=%s", commit),
		"--build-arg", fmt.Sprintf("BUILD_DATE=%s", date),
	)

	// Add tags
	for _, tag := range tags {
		dockerCmd = append(dockerCmd, "-t", tag)
	}

	// Add file
	dockerCmd = append(dockerCmd, "-f", *file)

	// Add platform if specified
	if *platform != "" {
		dockerCmd = append(dockerCmd, "--platform", *platform)
	}

	// Add context
	dockerCmd = append(dockerCmd, ".")

	// Log and run docker build
	logCommand("docker", dockerCmd...)

	cmd := exec.Command("docker", dockerCmd...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logError(fmt.Sprintf("docker build failed: %v", err))
		return err
	}

	logSuccess(fmt.Sprintf("Built %s:%s", imageName, version))

	// Push if --release is set and not dry-run
	if *release {
		if plan.DryRun {
			logWarning("Dry run - skipping push")
		} else {
			for _, tag := range tags {
				pushCmd := []string{"push", tag}
				logCommand("docker", pushCmd...)

				cmd := exec.Command("docker", pushCmd...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				if err := cmd.Run(); err != nil {
					logError(fmt.Sprintf("docker push failed: %v", err))
					return err
				}

				logSuccess(fmt.Sprintf("Pushed %s", tag))
			}
		}
	}

	logOutputs(map[string]string{
		"image":   imageName,
		"version": version,
		"tags":    strings.Join(tags, ","),
	})

	return nil
}

// runChecksums generates checksums for artifacts
func runChecksums(args []string) error {
	fs := newFlagSet("checksums")
	algorithm := fs.String("algorithm", "sha256", "Hash algorithm (sha256, sha512)")
	output := fs.String("output", "checksums.txt", "Output file")

	parseFlagsOrExit(fs, args)

	artifacts := fs.Args()
	if len(artifacts) == 0 {
		return fmt.Errorf("no artifacts specified")
	}

	logInputs(map[string]string{
		"algorithm": *algorithm,
		"artifacts": strings.Join(artifacts, ", "),
		"output":    *output,
	})

	// Expand globs
	var files []string
	for _, pattern := range artifacts {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", pattern, err)
		}
		files = append(files, matches...)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files matched patterns")
	}

	// Run checksum command based on algorithm
	var cmd *exec.Cmd
	switch *algorithm {
	case "sha256":
		cmd = exec.Command("sha256sum", files...)
	case "sha512":
		cmd = exec.Command("sha512sum", files...)
	default:
		return fmt.Errorf("unsupported algorithm: %s", *algorithm)
	}

	logCommand(cmd.Path, cmd.Args[1:]...)

	// Capture output
	outputBytes, err := cmd.Output()
	if err != nil {
		logError(fmt.Sprintf("checksum failed: %v", err))
		return err
	}

	// Write to file
	if err := os.WriteFile(*output, outputBytes, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", *output, err)
	}

	logSuccess(fmt.Sprintf("Generated %s", *output))

	logOutputs(map[string]string{
		"file":  *output,
		"count": fmt.Sprintf("%d", len(files)),
	})

	return nil
}

// runGitHubRelease creates a GitHub release with artifacts
func runGitHubRelease(args []string) error {
	fs := newFlagSet("github-release")
	title := fs.String("title", "", "Release title (default: auto-generated)")
	notes := fs.String("notes", "", "Release notes (default: auto-generated from git log)")
	prerelease := fs.Bool("prerelease", false, "Mark as pre-release")
	draft := fs.Bool("draft", false, "Create as draft")

	// Parse flags
	if err := fs.Parse(args); err != nil {
		return err
	}

	artifacts := fs.Args()

	// Load plan.json
	plan, err := loadPlan("plan.json")
	if err != nil {
		return fmt.Errorf("failed to load plan.json: %w", err)
	}

	// Get version
	version := plan.TagRelease
	if version == "" {
		version = plan.TagNext
	}
	if version == "" {
		return fmt.Errorf("no version in plan.json")
	}

	logInputs(map[string]string{
		"version":    version,
		"artifacts":  strings.Join(artifacts, ", "),
		"prerelease": fmt.Sprintf("%t", *prerelease),
		"draft":      fmt.Sprintf("%t", *draft),
		"dry_run":    fmt.Sprintf("%t", plan.DryRun),
	})

	// Check for GitHub token
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = os.Getenv("GH_TOKEN")
	}
	logEnv([]string{"GITHUB_TOKEN", "GH_TOKEN"})

	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN or GH_TOKEN environment variable required")
	}

	if plan.DryRun {
		logWarning("Dry run - skipping release creation")
		return nil
	}

	// Build gh release command
	ghCmd := []string{"release", "create", version}

	// Add title
	if *title != "" {
		ghCmd = append(ghCmd, "--title", *title)
	} else {
		// Get repo name from git
		repo := getRepoName()
		ghCmd = append(ghCmd, "--title", fmt.Sprintf("%s %s", repo, version))
	}

	// Add notes (auto-generate if not provided)
	if *notes != "" {
		ghCmd = append(ghCmd, "--notes", *notes)
	} else {
		ghCmd = append(ghCmd, "--generate-notes")
	}

	// Add flags
	if *prerelease {
		ghCmd = append(ghCmd, "--prerelease")
	}
	if *draft {
		ghCmd = append(ghCmd, "--draft")
	}

	// Add artifacts
	ghCmd = append(ghCmd, artifacts...)

	// Log and run
	logCommand("gh", ghCmd...)

	cmd := exec.Command("gh", ghCmd...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logError(fmt.Sprintf("gh release create failed: %v", err))
		return err
	}

	logSuccess(fmt.Sprintf("Created release %s", version))

	logOutputs(map[string]string{
		"version":   version,
		"artifacts": fmt.Sprintf("%d", len(artifacts)),
	})

	return nil
}

// runGitHubChangelog generates changelog from git history
func runGitHubChangelog(args []string) error {
	// Load plan.json
	plan, err := loadPlan("plan.json")
	if err != nil {
		return fmt.Errorf("failed to load plan.json: %w", err)
	}

	// Get version range
	latest := plan.TagLatest
	if latest == "" {
		latest = "HEAD"
	}

	// Generate changelog using git log
	cmd := exec.Command("git", "log", "--pretty=format:- %s (%h)", fmt.Sprintf("%s..HEAD", latest))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to generate changelog: %w", err)
	}

	fmt.Println(string(output))
	return nil
}

// getCurrentCommit gets the current git commit hash
func getCurrentCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// getCurrentDate gets the current date in ISO 8601 format
func getCurrentDate() string {
	cmd := exec.Command("date", "-u", "+%Y-%m-%dT%H:%M:%SZ")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// getRepoName gets the repository name from git
func getRepoName() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "project"
	}

	// Parse repo name from URL
	url := strings.TrimSpace(string(output))
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")
	// Get last part
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "project"
}
