package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// writeBoolOutput writes a boolean value as PublishTrue or PublishFalse
func writeBoolOutput(githubOutput, key string, value bool) {
	if value {
		writeOutput(githubOutput, key, PublishTrue)
	} else {
		writeOutput(githubOutput, key, PublishFalse)
	}
}

func runPlan(args []string) error {
	fs := newFlagSet("plan")

	// Version flags
	bump := fs.String("bump", "", "patch|minor|major (optional, auto-detect from commits if empty)")
	nextTag := fs.String("next-tag", "", "next release tag (optional, skip version computation if provided)")
	latestTag := fs.String("latest-tag", "", "latest release tag (optional, used with -next-tag)")

	// Policy flags
	mode := fs.String("mode", DefaultMode, "policy mode: release|rerelease|docker|goreleaser")
	image := fs.String("image", DefaultImage, "docker image repository")
	owner := fs.String("owner", "", "repository owner (optional, auto-detected from git if empty)")
	repo := fs.String("repo", "", "repository name (optional, auto-detected from git if empty)")
	sha := fs.String("sha", "", "git sha used for summary output")
	requiredSecrets := fs.String("required-secrets", "", "comma-separated required secret names (auto-detected by mode if empty)")
	resolveLatestTag := fs.Bool("resolve-latest-tag", false, "resolve latest tag from git (used by rerelease mode)")

	// Workflow control flags (for precalculating job execution)
	dryRun := fs.Bool("dry-run", false, "dry run mode")
	useNpm := fs.Bool("use-npm", true, "enable npm jobs")
	useMaven := fs.Bool("use-maven", true, "enable maven jobs")
	useDocker := fs.Bool("use-docker", true, "enable docker jobs")
	useGo := fs.Bool("use-go", true, "enable go jobs")
	useGoreleaser := fs.Bool("use-goreleaser", true, "enable goreleaser job")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Trim all flag values once for cleaner code
	modeVal := strings.TrimSpace(*mode)
	imageVal := strings.TrimSpace(*image)
	ownerVal := strings.TrimSpace(*owner)
	repoVal := strings.TrimSpace(*repo)
	shaVal := strings.TrimSpace(*sha)
	nextTagVal := strings.TrimSpace(*nextTag)
	latestTagVal := strings.TrimSpace(*latestTag)
	secretsVal := strings.TrimSpace(*requiredSecrets)

	// Log all inputs
	logInputs(map[string]string{
		"mode":             modeVal,
		"bump":             strings.TrimSpace(*bump),
		"next-tag":         nextTagVal,
		"latest-tag":       latestTagVal,
		"image":            imageVal,
		"owner":            ownerVal,
		"repo":             repoVal,
		"sha":              shaVal,
		"required-secrets": secretsVal,
		"dry-run":          fmt.Sprintf("%v", *dryRun),
		"use-npm":          fmt.Sprintf("%v", *useNpm),
		"use-maven":        fmt.Sprintf("%v", *useMaven),
		"use-docker":       fmt.Sprintf("%v", *useDocker),
		"use-go":           fmt.Sprintf("%v", *useGo),
		"use-goreleaser":   fmt.Sprintf("%v", *useGoreleaser),
	})

	// Auto-enable resolve-latest-tag for rerelease mode
	if modeVal == ModeRerelease && !*resolveLatestTag {
		*resolveLatestTag = true
	}

	fmt.Println("::group::Detect")

	// Detect project types
	detectedProjects := detectProjectTypes()
	_ = detectedProjects // Will be used to steer plan logic

	// Check if custom .goreleaser.yml exists
	hasCustomGoreleaserConfig := fileExists(FileGoReleaser) || fileExists(".goreleaser.yaml")
	if hasCustomGoreleaserConfig {
		fmt.Fprintln(os.Stderr, "  🚀 Using custom .goreleaser.yml config")
	}

	// Check if GoReleaser will handle Docker builds
	hasGoreleaserDocker := fileExists(FileGoreleaserContainerfile) || fileExists(FileGoreleaserDockerfile)

	// Check if standalone Docker files exist (not handled by GoReleaser)
	hasStandaloneDocker := fileExists(FileContainerfile) || fileExists(FileDockerfile)

	// Check if Go project (go.mod exists)
	hasGo := fileExists(FileGo)

	// Check if Maven project (pom.xml exists)
	hasMaven := fileExists(FileMaven)

	// Check if npm project (package.json exists)
	hasNpm := fileExists(FileNode)

	// Compute docker image if not provided or is default
	dockerImage := imageVal
	if dockerImage == "" || dockerImage == DefaultImage {
		// Compute from owner/repo
		ownerStr := ownerVal
		repoStr := repoVal

		// Try to get from environment if not provided
		if ownerStr == "" {
			ownerStr = os.Getenv("GITHUB_REPOSITORY_OWNER")
		}
		if repoStr == "" {
			// GITHUB_REPOSITORY is "owner/repo"
			fullRepo := os.Getenv("GITHUB_REPOSITORY")
			if fullRepo != "" {
				parts := strings.Split(fullRepo, "/")
				if len(parts) == 2 {
					repoStr = parts[1]
					if ownerStr == "" {
						ownerStr = parts[0]
					}
				}
			}
		}

		if ownerStr != "" && repoStr != "" {
			dockerImage = fmt.Sprintf("%s/%s", ownerStr, repoStr)
		} else if dockerImage == "" {
			dockerImage = DefaultImage
		}
	}

	fmt.Fprintf(os.Stderr, "  mode=%s\n", modeVal)
	fmt.Println("::endgroup::")

	// Set default required secrets based on mode if not provided
	secrets := secretsVal
	if secrets == "" {
		switch modeVal {
		case ModeGoreleaser:
			secrets = EnvHomebrewTapToken
		case ModeRerelease:
			// Only require Docker secrets if Docker is actually present
			if hasStandaloneDocker || hasGoreleaserDocker {
				secrets = fmt.Sprintf("%s,%s,%s", EnvDockerHubUsername, EnvDockerHubToken, EnvHomebrewTapToken)
			} else {
				secrets = EnvHomebrewTapToken
			}
		case ModeDocker:
			// Docker mode always needs Docker credentials
			secrets = fmt.Sprintf("%s,%s", EnvDockerHubUsername, EnvDockerHubToken)
		case ModeRelease:
			// Only require Docker secrets if Docker is actually present
			if hasStandaloneDocker || hasGoreleaserDocker {
				secrets = fmt.Sprintf("%s,%s", EnvDockerHubUsername, EnvDockerHubToken)
			}
			// else: empty string, no secrets required
		default:
			// Unknown mode, don't require secrets
			secrets = ""
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
	if nextTagVal != "" {
		// Use provided version (docker mode when called from release workflow)
		next = nextTagVal
		latest = latestTagVal
		if latest == "" {
			// Try to get latest from git if not provided
			latest, _ = git.GetLatestTag()
		}
		publish = PublishTrue
		fmt.Fprintf(os.Stderr, "📌 Using provided tag: %s\n", next)
	} else if modeVal == ModeRerelease {
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

	// Compute clean version (strip 'v' prefix)
	versionClean := next
	if strings.HasPrefix(versionClean, "v") {
		versionClean = versionClean[1:]
	}

	// Check if tag already exists
	tagExists := PublishFalse
	if next != "" {
		exists, err := git.TagExists(next)
		if err == nil && exists {
			tagExists = PublishTrue
			fmt.Fprintf(os.Stderr, "⚠️  Tag %s already exists\n", next)
		}
	}

	// Write version outputs
	writeOutput(githubOutput, OutputTagLatest, latest)
	if next != "" {
		writeOutput(githubOutput, OutputTagNext, next)
	}
	writeOutput(githubOutput, OutputTagExists, tagExists)
	writeOutput(githubOutput, OutputPublish, publish)

	// If we're skipping, output that and stop
	if publish == PublishSkip {
		writeOutput(githubOutput, OutputSkip, PublishTrue)
		writeOutput(githubOutput, OutputSummaryMessage, "Info: No release markers found. Skipping tag creation and publish.")
		fmt.Println("Info: No release markers found. Skipping release.")
		return nil
	}

	fmt.Println("::group::Policy")

	// Step 2: Compute release policy
	input := PolicyInput{
		Mode:            modeVal,
		EventName:       eventName,
		Publish:         publish,
		LatestTag:       latest,
		NextTag:         next,
		Image:           imageVal,
		SHA:             shaVal,
		RequiredSecrets: parseCSV(secrets),
		ResolveLatest:   *resolveLatestTag,
	}

	policy, err := computeReleasePolicy(input, &EnvProviderReal{}, git)
	if err != nil {
		return err
	}

	// Write policy outputs
	if policy.Skip != "" {
		writeOutput(githubOutput, OutputSkip, policy.Skip)
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
	writeBoolOutput(githubOutput, OutputGoreleaserYmlCurrent, hasCustomGoreleaserConfig)

	// Output goreleaser docker status
	writeBoolOutput(githubOutput, OutputGoreleaserDocker, hasGoreleaserDocker)

	// Output standalone docker status
	writeBoolOutput(githubOutput, OutputHasDocker, hasStandaloneDocker)

	// Output go project status
	writeBoolOutput(githubOutput, OutputHasGo, hasGo)

	// Output maven project status
	writeBoolOutput(githubOutput, OutputHasMaven, hasMaven)

	// Output npm project status
	writeBoolOutput(githubOutput, OutputHasNpm, hasNpm)

	// Output tag_latest flag (true unless rerelease mode)
	writeBoolOutput(githubOutput, OutputTagLatestFlag, modeVal != ModeRerelease)

	// Output computed docker image
	writeOutput(githubOutput, OutputDockerImage, dockerImage)

	// Output computed clean version (without 'v' prefix)
	if versionClean != "" {
		writeOutput(githubOutput, OutputVersionClean, versionClean)
	}

	// Precalculate "should run" decisions (testable logic in Go)
	skip := policy.Skip == PublishTrue
	shouldRunNpmBuild := !skip && hasNpm && *useNpm
	shouldRunGoBuild := !skip && hasGo && *useGo
	shouldRunMavenBuild := !skip && hasMaven && *useMaven
	shouldRunDockerBuild := !skip && !hasGoreleaserDocker && hasStandaloneDocker && *useDocker

	// Output should_run_* decisions
	writeBoolOutput(githubOutput, "should_build_npm", shouldRunNpmBuild)
	writeBoolOutput(githubOutput, "should_build_go", shouldRunGoBuild)
	writeBoolOutput(githubOutput, "should_build_maven", shouldRunMavenBuild)
	writeBoolOutput(githubOutput, "should_build_docker", shouldRunDockerBuild)

	// Determine tag for plan data
	tag := next
	if tag == "" {
		tag = policy.ReleaseTag
	}

	// Write plan data to JSON file for downstream jobs
	planData := map[string]interface{}{
		"mode":                      modeVal,
		"tool_ref":                  "", // Will be set by workflow
		"skip":                      skip,
		"tag":                       tag,
		"tag_exists":                tagExists == PublishTrue,
		"version":                   versionClean,
		"docker_image":              dockerImage,
		"has_go":                    hasGo,
		"has_docker":                hasStandaloneDocker,
		"has_maven":                 hasMaven,
		"has_npm":                   hasNpm,
		"goreleaser_docker":         hasGoreleaserDocker,
		"goreleaser_config_current": hasCustomGoreleaserConfig,
	}

	planJSON, err := json.MarshalIndent(planData, "", "  ")
	if err == nil {
		if err := os.WriteFile("/tmp/plan.json", planJSON, 0644); err == nil {
			fmt.Fprintln(os.Stderr, "  📝 Wrote plan data to /tmp/plan.json")
		} else {
			fmt.Fprintf(os.Stderr, "  ⚠️  Warning: Failed to write /tmp/plan.json: %v\n", err)
		}
	}

	fmt.Println("::endgroup::")

	// Handle Docker login for docker mode
	if modeVal == ModeDocker && policy.Skip != PublishTrue {
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

	// Check for GoReleaser config in release/rerelease modes
	if (modeVal == ModeRelease || modeVal == ModeRerelease) && policy.Skip != PublishTrue {
		fmt.Println("::group::GoReleaser config")
		hasCustomConfig := fileExists(FileGoReleaser) || fileExists(".goreleaser.yaml")
		if hasCustomConfig {
			writeOutput(githubOutput, OutputGoreleaserYmlCurrent, PublishTrue)
			fmt.Fprintln(os.Stderr, "  🚀 Using .goreleaser.yml config")
		} else {
			fmt.Fprintln(os.Stderr, "  ⚠️  No .goreleaser.yml found - goreleaser will use defaults or fail")
			writeOutput(githubOutput, OutputGoreleaserYmlCurrent, PublishFalse)
		}
		fmt.Println("::endgroup::")
	}

	// Print summary with visual diagram
	printReleaseDiagram(modeVal, latest, tag, policy.Skip == PublishTrue, hasGoreleaserDocker, hasCustomGoreleaserConfig)

	if policy.Message != "" {
		fmt.Printf("\n%s\n", policy.Message)
	}

	return nil
}
