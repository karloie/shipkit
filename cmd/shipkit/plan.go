package main

import (
	"fmt"
	"os"
	"strings"
)

func runPlan(args []string) error {
	fs := newFlagSet("plan")

	// Version flags
	bump := fs.String("bump", "", "patch|minor|major (optional, auto-detect from commits if empty)")
	nextTag := fs.String("next-tag", "", "next release tag (optional, skip version computation if provided)")
	latestTag := fs.String("latest-tag", "", "latest release tag (optional, used with -next-tag)")

	// Policy flags
	mode := fs.String("mode", DefaultMode, "policy mode: release|rerelease|docker|goreleaser")
	image := fs.String("image", DefaultImage, "docker image repository")
	sha := fs.String("sha", "", "git sha used for summary output")
	requiredSecrets := fs.String("required-secrets", "", "comma-separated required secret names (auto-detected by mode if empty)")
	resolveLatestTag := fs.Bool("resolve-latest-tag", false, "resolve latest tag from git (used by rerelease mode)")

	// GoReleaser config generation flags
	projectName := fs.String("project", "", "Project name for goreleaser config generation")
	binaryName := fs.String("binary", "", "Binary name for goreleaser config generation")
	repoOwner := fs.String("owner", "", "Repository owner for goreleaser config generation")
	repoName := fs.String("repo", "", "Repository name for goreleaser config generation")
	description := fs.String("description", "Application built with Go", "Project description for goreleaser config")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("::group::Detect")

	// Auto-detect project name from go.mod if not provided
	if *projectName == "" {
		*projectName = detectProjectName()
	}

	// Auto-detect binary name (defaults to project name)
	if *binaryName == "" {
		*binaryName = *projectName
	}

	// Auto-detect project description if using default value
	if *description == "Application built with Go" {
		if det := detectProjectDescription(); det != "" {
			*description = det
		}
	}

	// Detect project types
	detectedProjects := detectProjectTypes()
	_ = detectedProjects // Will be used to steer plan logic

	// Check if custom .goreleaser.yml exists
	hasCustomGoreleaserConfig := fileExists(FileGoReleaser) || fileExists(".goreleaser.yaml")
	if hasCustomGoreleaserConfig {
		fmt.Fprintln(os.Stderr, "🚀 Using custom .goreleaser.yml config")
	}

	// Check if GoReleaser will handle Docker builds
	hasGoreleaserDocker := fileExists(FileGoreleaserContainerfile) || fileExists(FileGoreleaserDockerfile)

	fmt.Fprintf(os.Stderr, "  project=%s  mode=%s\n", *projectName, strings.TrimSpace(*mode))
	fmt.Println("::endgroup::")

	// Set default required secrets based on mode if not provided
	secrets := strings.TrimSpace(*requiredSecrets)
	if secrets == "" {
		switch strings.TrimSpace(*mode) {
		case ModeGoreleaser:
			secrets = EnvHomebrewTapToken
		case ModeRerelease:
			secrets = fmt.Sprintf("%s,%s,%s", EnvDockerHubUsername, EnvDockerHubToken, EnvHomebrewTapToken)
		case ModeRelease, ModeDocker:
			secrets = fmt.Sprintf("%s,%s", EnvDockerHubUsername, EnvDockerHubToken)
		default:
			secrets = fmt.Sprintf("%s,%s", EnvDockerHubUsername, EnvDockerHubToken)
		}
	}

	eventName := os.Getenv(EnvGitHubEventName)
	githubOutput := os.Getenv(EnvGitHubOutput)
	token := os.Getenv(EnvGitHubToken)

	git := &GitProviderReal{}
	pr := &PRProviderReal{token: token}

	var latest, next, publish string
	var err error

	fmt.Println("::group::Version")

	// Step 1: Compute or use provided version
	if strings.TrimSpace(*nextTag) != "" {
		// Use provided version (docker mode when called from release workflow)
		next = strings.TrimSpace(*nextTag)
		latest = strings.TrimSpace(*latestTag)
		if latest == "" {
			// Try to get latest from git if not provided
			latest, _ = git.GetLatestTag()
		}
		publish = PublishTrue
		fmt.Fprintf(os.Stderr, "📌 Using provided tag: %s\n", next)
	} else if strings.TrimSpace(*mode) == ModeRerelease {
		// Rerelease resolves the tag itself — skip commit-based version computation
		publish = PublishTrue
		fmt.Fprintln(os.Stderr, "  Tag will be resolved from latest git tag")
	} else {
		// Compute version from git/commits
		latest, next, publish, err = computeVersion(eventName, *bump, git, pr)
		if err != nil {
			return err
		}
		if publish == PublishSkip {
			fmt.Fprintln(os.Stderr, "  No release markers found — will skip")
		} else {
			fmt.Fprintf(os.Stderr, "  %s → %s\n", latest, next)
		}
	}

	fmt.Println("::endgroup::")

	// Write version outputs
	writeOutput(githubOutput, OutputTagLatest, latest)
	if next != "" {
		writeOutput(githubOutput, OutputTagNext, next)
	}
	writeOutput(githubOutput, OutputPublish, publish)

	// If we're skipping, output that and stop
	if publish == PublishSkip {
		writeOutput(githubOutput, OutputDryRun, PublishTrue)
		writeOutput(githubOutput, OutputSummaryMessage, "Info: No release markers found. Skipping tag creation and publish.")
		fmt.Println("Info: No release markers found. Skipping release.")
		return nil
	}

	fmt.Println("::group::Policy")

	// Step 2: Compute release policy
	input := PolicyInput{
		Mode:            strings.TrimSpace(*mode),
		EventName:       eventName,
		Publish:         publish,
		LatestTag:       latest,
		NextTag:         next,
		Image:           strings.TrimSpace(*image),
		SHA:             strings.TrimSpace(*sha),
		RequiredSecrets: parseCSV(secrets),
		ResolveLatest:   *resolveLatestTag,
	}

	policy, err := computeReleasePolicy(input, &EnvProviderReal{}, git)
	if err != nil {
		return err
	}

	// Write policy outputs
	if policy.DryRun != "" {
		writeOutput(githubOutput, OutputDryRun, policy.DryRun)
	}
	if policy.Version != "" {
		writeOutput(githubOutput, OutputVersion, policy.Version)
	}
	if policy.VersionMajorMinor != "" {
		writeOutput(githubOutput, OutputVersionMajorMinor, policy.VersionMajorMinor)
	}
	if policy.Dockerfile != "" {
		writeOutput(githubOutput, OutputDockerfile, policy.Dockerfile)
	}
	if policy.ReleaseTag != "" {
		writeOutput(githubOutput, OutputReleaseTag, policy.ReleaseTag)
	}
	writeOutput(githubOutput, OutputSummaryMessage, policy.Message)

	// Output custom goreleaser config status
	if hasCustomGoreleaserConfig {
		writeOutput(githubOutput, OutputGoreleaserYmlCurrent, PublishTrue)
	} else {
		writeOutput(githubOutput, OutputGoreleaserYmlCurrent, PublishFalse)
	}

	// Output goreleaser docker status
	if hasGoreleaserDocker {
		writeOutput(githubOutput, OutputGoreleaserDocker, PublishTrue)
	} else {
		writeOutput(githubOutput, OutputGoreleaserDocker, PublishFalse)
	}

	fmt.Println("::endgroup::")

	// Handle Docker login for docker mode
	if strings.TrimSpace(*mode) == ModeDocker && policy.DryRun != PublishTrue {
		username := os.Getenv(EnvDockerHubUsername)
		dockerToken := os.Getenv(EnvDockerHubToken)

		if username != "" && dockerToken != "" {
			if err := dockerLogin(username, dockerToken); err != nil {
				return fmt.Errorf("docker login failed: %w", err)
			}
			writeOutput(githubOutput, "push", PublishTrue)
		} else {
			fmt.Fprintln(os.Stderr, "⚠️  Warning: Missing DockerHub credentials - will build locally without pushing")
			writeOutput(githubOutput, "push", PublishFalse)
		}
	}

	// Create tag for release mode when not in dryrun
	if strings.TrimSpace(*mode) == ModeRelease && policy.DryRun != PublishTrue && next != "" {
		if err := createGitTag(next); err != nil {
			return fmt.Errorf("failed to create tag: %w", err)
		}
	}

	// Generate GoReleaser config for release/rerelease modes when not in dryrun
	if (strings.TrimSpace(*mode) == ModeRelease || strings.TrimSpace(*mode) == ModeRerelease) && policy.DryRun != PublishTrue {
		fmt.Println("::group::GoReleaser config")
		hasCustomConfig := fileExists(FileGoReleaser) || fileExists(".goreleaser.yaml")
		if hasCustomConfig {
			writeOutput(githubOutput, OutputGoreleaserYmlCurrent, PublishTrue)
			fmt.Fprintln(os.Stderr, "🚀 Using custom .goreleaser.yml config")
		} else {
			writeOutput(githubOutput, OutputGoreleaserYmlCurrent, PublishFalse)
			fmt.Fprintln(os.Stderr, "  Generating GoReleaser config...")

			if *projectName != "" && *repoOwner != "" {
				configPath := ".goreleaser-generated.yml"

				// Repo name defaults to project name if not specified
				if *repoName == "" {
					*repoName = *projectName
				}

				mainPath := fmt.Sprintf("./cmd/%s", *projectName)
				dockerImage := fmt.Sprintf("%s/%s", *repoOwner, *projectName)

				detected := detectProjectTypes()
				hasNodeJS := hasProjectType(detected, "Node")
				hasChangelog := fileExists(FileChangelog)
				hasGoreleaserDocker := hasProjectType(detected, "GoReleaser Docker")

				// Get specific Docker file if GoReleaser Docker is detected
				dockerFile := ""
				if hasGoreleaserDocker {
					if fileExists(FileGoreleaserContainerfile) {
						dockerFile = FileGoreleaserContainerfile
					} else {
						dockerFile = FileGoreleaserDockerfile
					}
				}

				config := GoReleaserConfig{
					ProjectName:  *projectName,
					BinaryName:   *binaryName,
					MainPath:     mainPath,
					RepoOwner:    *repoOwner,
					RepoName:     *repoName,
					Description:  *description,
					License:      DefaultLicense,
					DockerImage:  dockerImage,
					HasNodeJS:    hasNodeJS,
					HasChangelog: hasChangelog,
					HasDocker:    hasGoreleaserDocker,
					DockerFile:   dockerFile,
				}

				if err := generateGoReleaserConfig(config, configPath); err != nil {
					return fmt.Errorf("failed to generate goreleaser config: %w", err)
				}

				writeOutput(githubOutput, OutputGoreleaserYmlNew, configPath)
				fmt.Fprintf(os.Stderr, "📝 Generated GoReleaser config at: %s\n", configPath)
			}
		}
		fmt.Println("::endgroup::")
	}

	// Print summary with visual diagram
	tag := next
	if tag == "" {
		tag = policy.ReleaseTag
	}
	printReleaseDiagram(strings.TrimSpace(*mode), latest, tag, policy.DryRun == PublishTrue, hasGoreleaserDocker, hasCustomGoreleaserConfig)

	if policy.Message != "" {
		fmt.Printf("\n%s\n", policy.Message)
	}

	return nil
}
