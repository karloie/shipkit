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
	// Log raw args BEFORE parsing (so we see them even if parsing fails)
	logInputs(map[string]string{
		"raw_args": strings.Join(args, " "),
	})

	fs := newFlagSet("plan")

	// Version flags
	bump := fs.String("bump", "", "Version bump")
	nextTag := fs.String("next-tag", "", "Next tag")
	latestTag := fs.String("latest-tag", "", "Latest tag")

	// Policy flags
	mode := fs.String("mode", DefaultMode, "Release mode")
	image := fs.String("image", DefaultImage, "Docker image")
	owner := fs.String("owner", "", "Repo owner")
	repo := fs.String("repo", "", "Repo name")
	sha := fs.String("sha", "", "Git SHA")
	requiredSecrets := fs.String("required-secrets", "", "Required secrets")
	resolveLatestTag := fs.Bool("resolve-latest-tag", false, "Resolve latest")

	// Workflow control flags (for precalculating job execution)
	dryRun := fs.Bool("dry-run", false, "Dry run")
	useNpm := fs.Bool("use-npm", true, "Enable npm")
	useMaven := fs.Bool("use-maven", true, "Enable maven")
	useDocker := fs.Bool("use-docker", true, "Enable docker")
	useGo := fs.Bool("use-go", true, "Enable go")
	useGoreleaser := fs.Bool("use-goreleaser", true, "Enable goreleaser")
	parseFlagsOrExit(fs, args)

	// Log all inputs RAW (before any processing)
	logInputs(map[string]string{
		"mode":             *mode,
		"bump":             *bump,
		"next-tag":         *nextTag,
		"latest-tag":       *latestTag,
		"image":            *image,
		"owner":            *owner,
		"repo":             *repo,
		"sha":              *sha,
		"required-secrets": *requiredSecrets,
		"dry-run":          fmt.Sprintf("%v", *dryRun),
		"use-npm":          fmt.Sprintf("%v", *useNpm),
		"use-maven":        fmt.Sprintf("%v", *useMaven),
		"use-docker":       fmt.Sprintf("%v", *useDocker),
		"use-go":           fmt.Sprintf("%v", *useGo),
		"use-goreleaser":   fmt.Sprintf("%v", *useGoreleaser),
	})

	// Trim all flag values once for cleaner code
	modeVal := strings.TrimSpace(*mode)
	imageVal := strings.TrimSpace(*image)
	ownerVal := strings.TrimSpace(*owner)
	repoVal := strings.TrimSpace(*repo)
	shaVal := strings.TrimSpace(*sha)
	nextTagVal := strings.TrimSpace(*nextTag)
	latestTagVal := strings.TrimSpace(*latestTag)
	secretsVal := strings.TrimSpace(*requiredSecrets)

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

	// Check for build orchestrators (Makefile, justfile, Taskfile)
	hasMakefile := fileExists("Makefile")
	hasJustfile := fileExists("justfile")
	hasTaskfile := fileExists("Taskfile.yml") || fileExists("Taskfile.yaml")

	// Determine build orchestrator preference
	buildOrchestrator := ""
	if hasMakefile {
		buildOrchestrator = "make"
	} else if hasJustfile {
		buildOrchestrator = "just"
	} else if hasTaskfile {
		buildOrchestrator = "task"
	} else {
		buildOrchestrator = "convention"
	}

	// Parse Makefile if it exists to understand available targets
	var makeTargets []string
	if hasMakefile {
		if graph, err := ParseMakefile("Makefile"); err == nil {
			makeTargets = graph.GetTargets()
			fmt.Fprintf(os.Stderr, "  📋 Detected %d Makefile targets\n", len(makeTargets))
		}
	}

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

	// If we're skipping, stop early (but write plan.json first for downstream)
	if publish == PublishSkip {
		fmt.Println("Info: No release markers found. Skipping release.")

		// Build minimal plan - same structure as full plan
		plan := map[string]string{
			"mode":            modeVal,
			"latest_tag":      latest,
			"next_tag":        latest,
			"tag":             latest,
			"tag_exists":      "false",
			"publish_skip":    "true",
			"summary_message": "Info: No release markers found. Skipping tag and publish.",
		}

		// Write plan.json for downstream jobs
		planJSON, _ := json.MarshalIndent(plan, "", "  ")
		if err := os.WriteFile("plan.json", planJSON, 0644); err == nil {
			fmt.Fprintln(os.Stderr, "  📝 Wrote plan data to plan.json")
		}

		// Output same plan to GITHUB_OUTPUT
		for key, value := range plan {
			writeOutput(githubOutput, key, value)
		}

		// Log outputs
		logOutputs(plan)
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
		DryRun:          *dryRun,
	}

	policy, err := computeReleasePolicy(input, &EnvProviderReal{}, git)
	if err != nil {
		return err
	}

	fmt.Println("::endgroup::")

	// Precalculate "should run" decisions (testable logic in Go)
	skip := policy.Skip == PublishTrue
	shouldRunNpmBuild := !skip && hasNpm && *useNpm
	shouldRunGoBuild := !skip && hasGo && *useGo
	shouldRunMavenBuild := !skip && hasMaven && *useMaven
	shouldRunDockerBuild := !skip && !hasGoreleaserDocker && hasStandaloneDocker && *useDocker

	// Determine tag for plan data
	tag := next
	if tag == "" {
		tag = policy.ReleaseTag
	}

	// Handle Docker login for docker mode
	pushDocker := ""
	if modeVal == ModeDocker && policy.Skip != PublishTrue {
		username := os.Getenv(EnvDockerHubUsername)
		dockerToken := os.Getenv(EnvDockerHubToken)

		if username != "" && dockerToken != "" {
			if err := dockerLogin(username, dockerToken); err != nil {
				return fmt.Errorf("docker login failed: %w", err)
			}
			pushDocker = PublishTrue
		} else {
			fmt.Fprintln(os.Stderr, "⚠️  Warning: Missing DockerHub credentials - will build locally without pushing")
			pushDocker = PublishFalse
		}
	}

	// Check for GoReleaser config in release/rerelease modes
	if (modeVal == ModeRelease || modeVal == ModeRerelease) && policy.Skip != PublishTrue {
		fmt.Println("::group::GoReleaser config")
		hasCustomGoreleaserConfig = fileExists(FileGoReleaser) || fileExists(".goreleaser.yaml")
		if hasCustomGoreleaserConfig {
			fmt.Fprintln(os.Stderr, "  🚀 Using .goreleaser.yml config")
		} else {
			fmt.Fprintln(os.Stderr, "  ⚠️  No .goreleaser.yml found - goreleaser will use defaults or fail")
		}
		fmt.Println("::endgroup::")
	}

	// Build complete plan map - SINGLE SOURCE OF TRUTH for everything
	plan := map[string]string{
		"mode":                   modeVal,
		"latest_tag":             latest,
		"next_tag":               next,
		"tag":                    tag,
		"tag_exists":             tagExists,
		"publish":                publish,
		"publish_skip":           fmt.Sprintf("%v", skip),
		"version":                policy.Version,
		"version_clean":          versionClean,
		"version_major_minor":    policy.VersionMajorMinor,
		"release_tag":            policy.ReleaseTag,
		"docker_image":           dockerImage,
		"dockerfile":             policy.Dockerfile,
		"build_orchestrator":     buildOrchestrator,
		"has_makefile":           fmt.Sprintf("%v", hasMakefile),
		"has_justfile":           fmt.Sprintf("%v", hasJustfile),
		"has_taskfile":           fmt.Sprintf("%v", hasTaskfile),
		"has_go":                 fmt.Sprintf("%v", hasGo),
		"has_docker":             fmt.Sprintf("%v", hasStandaloneDocker),
		"has_maven":              fmt.Sprintf("%v", hasMaven),
		"has_npm":                fmt.Sprintf("%v", hasNpm),
		"goreleaser_docker":      fmt.Sprintf("%v", hasGoreleaserDocker),
		"goreleaser_yml_current": fmt.Sprintf("%v", hasCustomGoreleaserConfig),
		"tag_latest":             fmt.Sprintf("%v", modeVal != ModeRerelease),
		"should_build_npm":       fmt.Sprintf("%v", shouldRunNpmBuild),
		"should_build_go":        fmt.Sprintf("%v", shouldRunGoBuild),
		"should_build_maven":     fmt.Sprintf("%v", shouldRunMavenBuild),
		"should_build_docker":    fmt.Sprintf("%v", shouldRunDockerBuild),
		"summary_message":        policy.Message,
	}
	if pushDocker != "" {
		plan["push"] = pushDocker
	}

	// Write plan.json for downstream jobs
	planJSON, err := json.MarshalIndent(plan, "", "  ")
	if err == nil {
		if err := os.WriteFile("plan.json", planJSON, 0644); err == nil {
			fmt.Fprintln(os.Stderr, "  📝 Wrote plan data to plan.json")
		} else {
			fmt.Fprintf(os.Stderr, "  ⚠️  Warning: Failed to write plan.json: %v\n", err)
		}
	}

	// Write all outputs to GITHUB_OUTPUT file for GitHub Actions
	for key, value := range plan {
		writeOutput(githubOutput, key, value)
	}

	// Print summary with visual diagram
	printReleaseDiagram(modeVal, latest, tag, policy.Skip == PublishTrue, hasGoreleaserDocker, hasCustomGoreleaserConfig)

	if policy.Message != "" {
		fmt.Printf("\n%s\n", policy.Message)
	}

	// Log outputs in human-readable format
	logOutputs(plan)

	return nil
}
