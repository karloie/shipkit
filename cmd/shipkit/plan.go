package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Plan struct {
	// Input fields (from CLI flags or plan.json)
	Bump                string `json:"bump"`
	DryRun              bool   `json:"dry_run,omitempty"`
	Mode                string `json:"mode"`
	Owner               string `json:"owner,omitempty"`
	Repo                string `json:"repo,omitempty"`
	RequiredSecrets     string `json:"required_secrets,omitempty"`
	ResolveLatestTag    bool   `json:"resolve_latest_tag,omitempty"`
	Sha                 string `json:"sha,omitempty"`
	UseDocker           bool   `json:"use_docker,omitempty"`
	UseGoreleaser       bool   `json:"use_goreleaser,omitempty"`
	UseGoreleaserDocker bool   `json:"use_goreleaser_docker,omitempty"`

	// Output fields (computed by runPlanClean)
	BuildOrchestrator    string              `json:"build_orchestrator"`
	BuildTargets         map[string][]string `json:"build_targets,omitempty"` // Target name → dependencies (empty array if no deps)
	DockerFile           string              `json:"docker_file"`
	DockerImage          string              `json:"docker_image"`
	DockerTagLatest      bool                `json:"docker_tag_latest"`
	GoreleaserConfig     string              `json:"goreleaser_config"`
	GoreleaserDockerfile string              `json:"goreleaser_dockerfile,omitempty"`
	HasDocker            bool                `json:"has_docker"`
	HasJustfile          bool                `json:"has_justfile"`
	HasMakefile          bool                `json:"has_makefile"`
	HasTaskfile          bool                `json:"has_taskfile"`
	ReleaseDocker        bool                `json:"release_docker,omitempty"`
	ReleaseSkip          bool                `json:"release_skip"`
	TagRelease           string              `json:"tag_release"` // Effective tag for this release (NextTag || ReleaseTag)
	TagExists            bool                `json:"tag_exists"`
	TagLatest            string              `json:"tag_latest"`
	TagNext              string              `json:"tag_next"`
	VersionClean         string              `json:"version_clean"`
	VersionMajorMinor    string              `json:"version_major_minor"`

	// Runtime results (from workflow execution)
	JobResults map[string]string `json:"job_results,omitempty"` // job_name -> "success"|"failure"|"skipped"
}

func runPlan(args []string) error {
	logInputs(map[string]string{
		"raw_args": strings.Join(args, " "),
	})

	// Load plan.json if it exists
	var plan Plan
	data, err := os.ReadFile("plan.json")
	if err == nil {
		json.Unmarshal(data, &plan)
	}

	// Parse CLI flags
	fs := newFlagSet("plan")

	// Version flags
	bump := fs.String("bump", plan.Bump, "Version bump")
	nextTag := fs.String("next-tag", plan.TagNext, "Next tag")
	latestTag := fs.String("latest-tag", plan.TagLatest, "Latest tag")

	// Policy flags
	mode := fs.String("mode", DefaultMode, "Release mode")
	image := fs.String("image", DefaultImage, "Docker image")
	owner := fs.String("owner", plan.Owner, "Repo owner")
	repo := fs.String("repo", plan.Repo, "Repo name")
	sha := fs.String("sha", plan.Sha, "Git SHA")
	requiredSecrets := fs.String("required-secrets", plan.RequiredSecrets, "Required secrets")
	resolveLatestTag := fs.Bool("resolve-latest-tag", plan.ResolveLatestTag, "Resolve latest")

	// Workflow control flags (for precalculating job execution)
	dryRun := fs.Bool("dry-run", plan.DryRun, "Dry run")
	useDocker := fs.Bool("use-docker", true, "Enable docker")
	useGoreleaser := fs.Bool("use-goreleaser", true, "Enable goreleaser")
	useGoreleaserDocker := fs.Bool("use-goreleaser-docker", false, "Enable goreleaser docker")

	parseFlagsOrExit(fs, args)

	// Override plan with CLI args
	plan.Bump = strings.TrimSpace(*bump)
	plan.TagNext = strings.TrimSpace(*nextTag)
	plan.TagLatest = strings.TrimSpace(*latestTag)
	plan.Mode = strings.TrimSpace(*mode)
	plan.DockerImage = strings.TrimSpace(*image)
	plan.Owner = strings.TrimSpace(*owner)
	plan.Repo = strings.TrimSpace(*repo)
	plan.Sha = strings.TrimSpace(*sha)
	plan.RequiredSecrets = strings.TrimSpace(*requiredSecrets)
	plan.ResolveLatestTag = *resolveLatestTag
	plan.DryRun = *dryRun
	plan.UseDocker = *useDocker
	plan.UseGoreleaser = *useGoreleaser
	plan.UseGoreleaserDocker = *useGoreleaserDocker

	// Log inputs
	logInputs(map[string]string{
		"mode":                  plan.Mode,
		"bump":                  plan.Bump,
		"next-tag":              plan.TagNext,
		"latest-tag":            plan.TagLatest,
		"image":                 plan.DockerImage,
		"owner":                 plan.Owner,
		"repo":                  plan.Repo,
		"sha":                   plan.Sha,
		"required-secrets":      plan.RequiredSecrets,
		"dry-run":               fmt.Sprintf("%v", plan.DryRun),
		"use-docker":            fmt.Sprintf("%v", plan.UseDocker),
		"use-goreleaser":        fmt.Sprintf("%v", plan.UseGoreleaser),
		"use-goreleaser-docker": fmt.Sprintf("%v", plan.UseGoreleaserDocker),
	})

	return runPlanClean(&plan, nil, nil)
}

// runPlanClean writes a Plan to outputs (plan.json, GITHUB_OUTPUT, logs)
// git and pr can be nil to use real implementations
func runPlanClean(plan *Plan, git GitProvider, pr PRProvider) error {
	githubOutput := os.Getenv(EnvGitHubOutput)

	// Auto-enable resolve-latest-tag for rerelease mode
	if plan.Mode == ModeRerelease && !plan.ResolveLatestTag {
		plan.ResolveLatestTag = true
	}

	// Detect project types
	detectedProjects := detectProjectTypes()
	_ = detectedProjects // Will be used to steer plan logic

	// Check if custom .goreleaser.yml exists and populate plan
	if fileExists(FileGoReleaser) {
		plan.GoreleaserConfig = FileGoReleaser
	} else if fileExists(".goreleaser.yaml") {
		plan.GoreleaserConfig = ".goreleaser.yaml"
	} else if plan.UseGoreleaser {
		// No config file found, will use autogenerated config
		plan.GoreleaserConfig = "/tmp/.goreleaser.yml"
	}

	// Detect Docker files and populate plan
	if fileExists(FileContainerfile) {
		plan.GoreleaserDockerfile = FileContainerfile
	} else if fileExists(FileDockerfile) {
		plan.GoreleaserDockerfile = FileDockerfile
	}

	// Check for build orchestrators (Makefile, justfile, Taskfile) and populate plan
	plan.HasMakefile = fileExists("Makefile")
	plan.HasJustfile = fileExists("justfile")
	plan.HasTaskfile = fileExists("Taskfile.yml") || fileExists("Taskfile.yaml")

	// Determine build orchestrator preference and populate plan
	if plan.HasMakefile {
		plan.BuildOrchestrator = "make"
	} else if plan.HasJustfile {
		plan.BuildOrchestrator = "just"
	} else if plan.HasTaskfile {
		plan.BuildOrchestrator = "task"
	} else {
		plan.BuildOrchestrator = "convention"
	}

	// Parse Makefile if it exists to extract ALL targets
	if plan.HasMakefile {
		if graph, err := ParseMakefile("Makefile"); err == nil {
			plan.BuildTargets = make(map[string][]string)

			// Extract ALL targets with their dependencies (empty array if none)
			targets := graph.GetTargets()
			for _, target := range targets {
				if t, exists := graph.Targets[target]; exists {
					if t.Dependencies == nil {
						plan.BuildTargets[target] = []string{}
					} else {
						plan.BuildTargets[target] = t.Dependencies
					}
				}
			}
		}
	}

	// Parse justfile if it exists to extract ALL recipes
	if plan.HasJustfile {
		if graph, err := ParseJustfile("justfile"); err == nil {
			plan.BuildTargets = make(map[string][]string)

			// Extract ALL recipes with their dependencies (empty array if none)
			recipes := graph.GetRecipes()
			for _, recipe := range recipes {
				if r, exists := graph.Recipes[recipe]; exists {
					if r.Dependencies == nil {
						plan.BuildTargets[recipe] = []string{}
					} else {
						plan.BuildTargets[recipe] = r.Dependencies
					}
				}
			}
		}
	}

	// Parse Taskfile if it exists to extract ALL tasks
	if plan.HasTaskfile {
		taskfilePath := "Taskfile.yml"
		if fileExists("Taskfile.yaml") {
			taskfilePath = "Taskfile.yaml"
		}
		if graph, err := ParseTaskfile(taskfilePath); err == nil {
			plan.BuildTargets = make(map[string][]string)

			// Extract ALL tasks with their dependencies (empty array if none)
			tasks := graph.GetTasks()
			for _, task := range tasks {
				if t, exists := graph.Tasks[task]; exists {
					if t.Dependencies == nil {
						plan.BuildTargets[task] = []string{}
					} else {
						plan.BuildTargets[task] = t.Dependencies
					}
				}
			}
		}
	}

	// Compute docker image if not provided or is default and populate plan
	if plan.DockerImage == "" || plan.DockerImage == DefaultImage {
		// Compute from owner/repo
		ownerStr := plan.Owner
		repoStr := plan.Repo

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
			plan.DockerImage = fmt.Sprintf("%s/%s", ownerStr, repoStr)
		} else if plan.DockerImage == "" {
			plan.DockerImage = DefaultImage
		}
	}

	// Set default required secrets based on mode if not provided
	secrets := plan.RequiredSecrets
	if secrets == "" {
		switch plan.Mode {
		case ModeGoreleaser:
			secrets = EnvHomebrewTapToken
		case ModeRerelease:
			// Only require Docker secrets if Docker is actually present
			if plan.GoreleaserDockerfile != "" {
				secrets = fmt.Sprintf("%s,%s,%s", EnvDockerHubUsername, EnvDockerHubToken, EnvHomebrewTapToken)
			} else {
				secrets = EnvHomebrewTapToken
			}
		case ModeDocker:
			// Docker mode always needs Docker credentials
			secrets = fmt.Sprintf("%s,%s", EnvDockerHubUsername, EnvDockerHubToken)
		case ModeRelease:
			// Only require Docker secrets if Docker is actually present
			if plan.GoreleaserDockerfile != "" {
				secrets = fmt.Sprintf("%s,%s", EnvDockerHubUsername, EnvDockerHubToken)
			}
			// else: empty string, no secrets required
		default:
			// Unknown mode, don't require secrets
			secrets = ""
		}
	}

	eventName := os.Getenv(EnvGitHubEventName)
	token := os.Getenv(EnvGitHubToken)

	// Use provided providers or default to real implementations
	if git == nil {
		git = &GitProviderReal{}
	}
	if pr == nil {
		pr = &PRProviderReal{token: token}
	}

	var release string
	var err error

	// Step 1: Compute or use provided version and populate plan
	if plan.TagNext != "" {
		// Use provided version (docker mode when called from release workflow)
		if plan.TagLatest == "" {
			// Try to get latest from git if not provided
			plan.TagLatest, _ = git.GetLatestTag()
		}
		release = ReleaseTrue
	} else if plan.Mode == ModeRerelease {
		// Rerelease resolves the tag itself — skip commit-based version computation
		release = ReleaseTrue
	} else {
		// Compute version from git/commits and populate plan
		plan.TagLatest, plan.TagNext, release, err = computeVersion(eventName, plan.Bump, git, pr)
		if err != nil {
			return outputPartialPlanOnError(githubOutput, plan, err)
		}
	}

	// Compute clean version (strip 'v' prefix) and populate plan
	plan.VersionClean = strings.TrimPrefix(plan.TagNext, "v")

	// Check if tag already exists and populate plan
	if plan.TagNext != "" {
		exists, err := git.TagExists(plan.TagNext)
		if err == nil && exists {
			plan.TagExists = true
		}
	}

	// If we're skipping, stop early (but write plan.json first for downstream)
	if release == ReleaseSkip {
		// Populate remaining plan fields for skip case
		plan.TagRelease = plan.TagLatest
		plan.ReleaseSkip = true
		plan.DockerTagLatest = plan.Mode != ModeRerelease
		plan.HasDocker = plan.GoreleaserDockerfile != ""

		// Write plan.json for downstream jobs
		planJSON, _ := json.MarshalIndent(plan, "", "  ")
		os.WriteFile("plan.json", planJSON, 0644)

		// Output to GITHUB_OUTPUT
		outputMap := plan.ToOutputMap()
		for key, value := range outputMap {
			writeOutput(githubOutput, key, value)
		}

		// Log plan as sorted JSON
		fmt.Fprintln(os.Stderr, "PLAN:")
		fmt.Fprintln(os.Stderr, string(planJSON))
		return nil
	}

	// Step 2: Compute release policy
	input := PolicyInput{
		Mode:            plan.Mode,
		EventName:       eventName,
		Release:         release,
		LatestTag:       plan.TagLatest,
		NextTag:         plan.TagNext,
		Image:           plan.DockerImage,
		SHA:             plan.Sha,
		RequiredSecrets: parseCSV(secrets),
		ResolveLatest:   plan.ResolveLatestTag,
		DryRun:          plan.DryRun,
	}

	policy, err := computeReleasePolicy(input, &EnvProviderReal{}, git)
	if err != nil {
		plan.TagRelease = plan.TagNext
		plan.ReleaseSkip = true
		plan.DockerTagLatest = plan.Mode != ModeRerelease
		plan.HasDocker = plan.GoreleaserDockerfile != ""
		return outputPartialPlanOnError(githubOutput, plan, err)
	}

	// Populate plan with policy results
	plan.ReleaseSkip = policy.Skip == ReleaseTrue
	plan.VersionMajorMinor = policy.VersionMajorMinor
	plan.DockerFile = policy.Dockerfile

	// Determine tag for plan data
	plan.TagRelease = plan.TagNext
	if plan.TagRelease == "" {
		plan.TagRelease = policy.ReleaseTag
	}

	// Handle Docker login for docker workflows
	hasDockerfile := policy.Dockerfile != "" || plan.GoreleaserDockerfile != ""
	if plan.UseDocker && !plan.ReleaseSkip && hasDockerfile {
		username := os.Getenv(EnvDockerHubUsername)
		dockerToken := os.Getenv(EnvDockerHubToken)

		if username != "" && dockerToken != "" {
			if err := dockerLogin(username, dockerToken); err != nil {
				plan.DockerTagLatest = plan.Mode != ModeRerelease
				plan.HasDocker = policy.Dockerfile != ""
				return outputPartialPlanOnError(githubOutput, plan, fmt.Errorf("docker login failed: %w", err))
			}
			plan.ReleaseDocker = true
		}
	}

	// Check for GoReleaser config in release/rerelease modes and populate plan
	if (plan.Mode == ModeRelease || plan.Mode == ModeRerelease) && !plan.ReleaseSkip && plan.UseGoreleaser {
		if fileExists(FileGoReleaser) {
			plan.GoreleaserConfig = FileGoReleaser
		} else if fileExists(".goreleaser.yaml") {
			plan.GoreleaserConfig = ".goreleaser.yaml"
		} else if plan.GoreleaserConfig == "" {
			// No config file found, will use autogenerated config
			plan.GoreleaserConfig = "/tmp/.goreleaser.yml"
		}
	}

	// Populate remaining plan fields
	plan.DockerTagLatest = plan.Mode != ModeRerelease
	plan.HasDocker = policy.Dockerfile != ""

	// Write plan.json for downstream jobs
	planJSON, err := json.MarshalIndent(plan, "", "  ")
	if err == nil {
		os.WriteFile("plan.json", planJSON, 0644)
	}

	// Write all outputs to GITHUB_OUTPUT file for GitHub Actions
	outputMap := plan.ToOutputMap()
	for key, value := range outputMap {
		writeOutput(githubOutput, key, value)
	}

	// Log plan as sorted JSON
	if planJSON != nil {
		fmt.Fprintln(os.Stderr, "PLAN:")
		fmt.Fprintln(os.Stderr, string(planJSON))
	}

	return nil
}

// ToOutputMap converts Plan to string map for GITHUB_OUTPUT
func (p *Plan) ToOutputMap() map[string]string {
	// Marshal to JSON then unmarshal to map[string]interface{}, then convert to string map
	jsonBytes, _ := json.Marshal(p)
	var rawMap map[string]interface{}
	json.Unmarshal(jsonBytes, &rawMap)

	// Convert all values to strings
	m := make(map[string]string, len(rawMap))
	for k, v := range rawMap {
		m[k] = fmt.Sprintf("%v", v)
	}
	return m
}

// outputPartialPlanOnError writes whatever plan state is available when an error occurs
func outputPartialPlanOnError(githubOutput string, p *Plan, err error) error {
	// Ensure ReleaseSkip is true on error
	p.ReleaseSkip = true

	// Write partial plan.json
	if planJSON, jsonErr := json.MarshalIndent(p, "", "  "); jsonErr == nil {
		os.WriteFile("plan.json", planJSON, 0644)

		// Write outputs
		outputMap := p.ToOutputMap()
		for key, value := range outputMap {
			writeOutput(githubOutput, key, value)
		}

		// Log plan as sorted JSON
		fmt.Fprintln(os.Stderr, "PLAN:")
		fmt.Fprintln(os.Stderr, string(planJSON))
	}

	return err
}

// writeBoolOutput writes a boolean value as ReleaseTrue or ReleaseFalse
func writeBoolOutput(githubOutput, key string, value bool) {
	if value {
		writeOutput(githubOutput, key, ReleaseTrue)
	} else {
		writeOutput(githubOutput, key, ReleaseFalse)
	}
}
