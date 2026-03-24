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
	fmt.Printf("export CONTAINER_IMAGE=%s\n", plan.DockerImage)
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
	file := fs.String("file", "", "Dockerfile to use (auto-detects Containerfile or Dockerfile)")
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

	// Get image name (defaults to plan.json)
	imageName := *image
	if imageName == "" {
		imageName = plan.DockerImage
	}
	if imageName == "" {
		return fmt.Errorf("no image specified (use --image or set docker_image in plan.json)")
	}

	// Auto-detect Dockerfile if not specified
	dockerFile := *file
	if dockerFile == "" {
		if _, err := os.Stat("Containerfile"); err == nil {
			dockerFile = "Containerfile"
		} else if _, err := os.Stat("Dockerfile"); err == nil {
			dockerFile = "Dockerfile"
		} else {
			return fmt.Errorf("no Containerfile or Dockerfile found (use --file to specify)")
		}
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
		"file":     dockerFile,
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
	dockerCmd = append(dockerCmd, "-f", dockerFile)

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
			// Docker login before pushing
			username := os.Getenv("DOCKERHUB_USERNAME")
			token := os.Getenv("DOCKERHUB_TOKEN")
			if username != "" && token != "" {
				logCommand("docker", "login", "-u", username, "-p", "***")
				loginCmd := exec.Command("docker", "login", "-u", username, "--password-stdin")
				loginCmd.Stdin = strings.NewReader(token)
				loginCmd.Stdout = os.Stdout
				loginCmd.Stderr = os.Stderr
				if err := loginCmd.Run(); err != nil {
					logError(fmt.Sprintf("docker login failed: %v", err))
					return err
				}
				logSuccess("Docker login successful")
			}

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

// runGoreleaserUtil wraps goreleaser with auto-generation support
func runGoreleaserUtil(args []string) error {
	fs := newFlagSet("goreleaser")
	generate := fs.Bool("generate", false, "Auto-generate .goreleaser.yml if missing")
	homebrew := fs.Bool("homebrew", false, "Include homebrew tap configuration")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Load plan.json
	plan, err := loadPlan("plan.json")
	if err != nil {
		return fmt.Errorf("failed to load plan.json: %w", err)
	}

	version := plan.TagRelease
	if version == "" {
		version = plan.TagNext
	}

	logInputs(map[string]string{
		"generate": fmt.Sprintf("%t", *generate),
		"homebrew": fmt.Sprintf("%t", *homebrew),
		"version":  version,
		"image":    plan.DockerImage,
	})

	// Check if .goreleaser.yml exists
	configPath := ".goreleaser.yml"
	_, err = os.Stat(configPath)
	configExists := err == nil

	// Generate config if requested and missing
	if *generate && !configExists {
		if err := generateGoreleaserConfig(plan, *homebrew); err != nil {
			return fmt.Errorf("failed to generate .goreleaser.yml: %w", err)
		}
		fmt.Println("✅ Generated .goreleaser.yml")
	}

	// Run goreleaser
	logCommand("goreleaser", "release", "--clean")
	cmd := exec.Command("goreleaser", "release", "--clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("goreleaser failed: %w", err)
	}

	logOutputs(map[string]string{
		"status": "success",
	})

	return nil
}

// generateGoreleaserConfig creates a minimal .goreleaser.yml
func generateGoreleaserConfig(plan *Plan, includeHomebrew bool) error {
	repoName := getRepoName()
	owner := getRepoOwner()

	config := fmt.Sprintf(`# GoReleaser configuration (generated by shipkit)
version: 2

# Build configuration
builds:
  - id: %s
    main: ./cmd/%s
    binary: %s
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.Version={{ .Version }}
      - -X main.GitCommit={{ .Commit }}
      - -X main.BuildTime={{ .Date }}

# Archives configuration
archives:
  - id: %s-archive
    builds:
      - %s
    format: binary
    name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"

# Checksum configuration
checksum:
  name_template: 'checksums.txt'

# GitHub release configuration
release:
  github:
    owner: %s
    name: %s
`, repoName, repoName, repoName, repoName, repoName, owner, repoName)

	// Add homebrew tap if requested
	if includeHomebrew {
		homebrewConfig := fmt.Sprintf(`
# Homebrew tap configuration
brews:
  - repository:
      owner: %s
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com
    directory: Casks
    homepage: "https://github.com/%s/%s"
    description: "Application built with Go"
    cask: |
      cask "%s" do
        name "%s"
        desc "Application built with Go"
        homepage "https://github.com/%s/%s"
        version "{{ .Version }}"
        
        on_macos do
          if Hardware::CPU.arm?
            url "https://github.com/%s/%s/releases/download/v{{ .Version }}/%s_darwin_arm64"
            sha256 "{{ .ArtifactChecksumFor \"%s_darwin_arm64\" }}"
          else
            url "https://github.com/%s/%s/releases/download/v{{ .Version }}/%s_darwin_amd64"
            sha256 "{{ .ArtifactChecksumFor \"%s_darwin_amd64\" }}"
          end
        end
        
        livecheck do
          skip "Auto-generated on release."
        end
        
        binary "%s_darwin_#{Hardware::CPU.arch}", target: "%s"
      end
`, owner, owner, repoName, repoName, repoName, owner, repoName, owner, repoName, repoName, repoName, owner, repoName, repoName, repoName, repoName, repoName)
		config += homebrewConfig
	}

	// Write config file
	return os.WriteFile(".goreleaser.yml", []byte(config), 0644)
}

// getRepoOwner gets the repository owner from git
func getRepoOwner() string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	// Parse owner from URL
	url := strings.TrimSpace(string(output))
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle github.com:owner/repo format
	if strings.Contains(url, ":") {
		parts := strings.Split(url, ":")
		if len(parts) >= 2 {
			pathParts := strings.Split(parts[1], "/")
			if len(pathParts) >= 2 {
				return pathParts[0]
			}
		}
	}

	// Handle https://github.com/owner/repo format
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}

	return "unknown"
}

// runInstall installs build tools (goreleaser, node, etc.)
func runInstall(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: shipkit install <tool>\nAvailable tools: goreleaser")
	}

	tool := args[0]

	logInputs(map[string]string{
		"tool": tool,
	})

	// Check if tool is already installed
	if path, err := exec.LookPath(tool); err == nil {
		fmt.Printf("✓ %s already installed: %s\n", tool, path)
		logOutputs(map[string]string{
			"status": "already_installed",
			"path":   path,
		})
		return nil
	}

	// Install based on tool
	fmt.Printf("📦 Installing %s...\n", tool)

	var cmd *exec.Cmd
	switch tool {
	case "goreleaser":
		// Check if go is available
		if _, err := exec.LookPath("go"); err != nil {
			return fmt.Errorf("go is not installed (required to install goreleaser)")
		}
		cmd = exec.Command("go", "install", "github.com/goreleaser/goreleaser@latest")

	default:
		return fmt.Errorf("unknown tool: %s\nAvailable tools: goreleaser", tool)
	}

	logCommand(cmd.Path, cmd.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logError(fmt.Sprintf("%s installation failed: %v", tool, err))
		return err
	}

	// Verify installation
	if path, err := exec.LookPath(tool); err == nil {
		logSuccess(fmt.Sprintf("Installed %s: %s", tool, path))
		logOutputs(map[string]string{
			"status": "installed",
			"path":   path,
		})
		return nil
	} else {
		// Installation succeeded but tool not in PATH
		// This can happen when go bin is not in PATH yet
		goPath := os.Getenv("GOPATH")
		if goPath == "" {
			goPath = filepath.Join(os.Getenv("HOME"), "go")
		}
		goBin := filepath.Join(goPath, "bin", tool)

		logWarning(fmt.Sprintf("%s installed but not in PATH. Add to PATH: %s", tool, filepath.Join(goPath, "bin")))
		logOutputs(map[string]string{
			"status":   "installed_not_in_path",
			"location": goBin,
		})
		return nil
	}
}
