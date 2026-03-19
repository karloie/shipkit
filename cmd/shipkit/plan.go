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

	// Policy flags
	mode := fs.String("mode", DefaultMode, "policy mode: release|rerelease|docker|goreleaser")
	image := fs.String("image", DefaultImage, "docker image repository")
	sha := fs.String("sha", "", "git sha used for summary output")
	requiredSecrets := fs.String("required-secrets", "", "comma-separated required secret names (auto-detected by mode if empty)")
	resolveLatestTag := fs.Bool("resolve-latest-tag", false, "resolve latest tag from git (used by rerelease mode)")

	if err := fs.Parse(args); err != nil {
		return err
	}

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

	// Step 1: Compute version
	latest, next, publish, err := computeVersion(eventName, *bump, git, pr)
	if err != nil {
		return err
	}

	// Write version outputs
	writeOutput(githubOutput, OutputLatestTag, latest)
	if next != "" {
		writeOutput(githubOutput, OutputNextTag, next)
	}
	writeOutput(githubOutput, OutputPublish, publish)

	// If we're skipping, output that and stop
	if publish == PublishSkip {
		writeOutput(githubOutput, OutputShouldPublish, PublishFalse)
		writeOutput(githubOutput, OutputSummaryMessage, "Info: No release markers found. Skipping tag creation and publish.")
		fmt.Println("Info: No release markers found. Skipping release.")
		return nil
	}

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
	if policy.ShouldPublish {
		writeOutput(githubOutput, OutputShouldPublish, PublishTrue)
	} else {
		writeOutput(githubOutput, OutputShouldPublish, PublishFalse)
	}
	if policy.PublishMode != "" {
		writeOutput(githubOutput, OutputPublishMode, policy.PublishMode)
	}
	if policy.DockerVersion != "" {
		writeOutput(githubOutput, OutputDockerVersion, policy.DockerVersion)
	}
	if policy.DockerMajorMinor != "" {
		writeOutput(githubOutput, OutputDockerMajorMinor, policy.DockerMajorMinor)
	}
	if policy.ReleaseTag != "" {
		writeOutput(githubOutput, OutputReleaseTag, policy.ReleaseTag)
	}
	writeOutput(githubOutput, OutputSummaryMessage, policy.Message)

	// Print summary
	fmt.Printf("🔄 Bumped from: %s\n", latest)
	fmt.Printf("🎉 Released new: %s\n", next)
	fmt.Printf("🚀 Should publish: %v\n", policy.ShouldPublish)

	return nil
}
