package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type ReleasePolicy struct {
	ShouldPublish    bool
	DockerVersion    string
	DockerMajorMinor string
	Dockerfile       string
	ReleaseTag       string
	PublishMode      string
	Message          string
}

type PolicyInput struct {
	Mode            string
	EventName       string
	Publish         string
	LatestTag       string
	NextTag         string
	Image           string
	SHA             string
	RequiredSecrets []string
	ResolveLatest   bool
}

func runPolicy(args []string) error {
	fs := newFlagSet("policy")
	mode := fs.String("mode", DefaultMode, "policy mode: release|rerelease|docker|goreleaser")
	eventName := fs.String("event-name", os.Getenv(EnvGitHubEventName), "workflow event name")
	publish := fs.String("publish", "", "publish output from version tool (true|skip)")
	latestTag := fs.String("latest-tag", "", "latest release tag, e.g. v1.2.2")
	nextTag := fs.String("next-tag", "", "next release tag, e.g. v1.2.3")
	image := fs.String("image", DefaultImage, "docker image repository")
	sha := fs.String("sha", "", "git sha used for summary output")
	requiredSecrets := fs.String("required-secrets", EnvDockerHubUsername+","+EnvDockerHubToken, "comma-separated required secret names")
	resolveLatestTag := fs.Bool("resolve-latest-tag", false, "resolve latest tag from git (used by rerelease mode)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	input := PolicyInput{
		Mode:            strings.TrimSpace(*mode),
		EventName:       strings.TrimSpace(*eventName),
		Publish:         strings.TrimSpace(*publish),
		LatestTag:       strings.TrimSpace(*latestTag),
		NextTag:         strings.TrimSpace(*nextTag),
		Image:           strings.TrimSpace(*image),
		SHA:             strings.TrimSpace(*sha),
		RequiredSecrets: parseCSV(*requiredSecrets),
		ResolveLatest:   *resolveLatestTag,
	}

	git := &GitProviderReal{}
	policy, err := computeReleasePolicy(input, &EnvProviderReal{}, git)
	if err != nil {
		return err
	}

	githubOutput := os.Getenv(EnvGitHubOutput)
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
	if policy.Dockerfile != "" {
		writeOutput(githubOutput, OutputDockerfile, policy.Dockerfile)
	}
	if policy.ReleaseTag != "" {
		writeOutput(githubOutput, OutputReleaseTag, policy.ReleaseTag)
	}
	writeOutput(githubOutput, OutputSummaryMessage, policy.Message)
	return nil
}

func computeReleasePolicy(input PolicyInput, env EnvProvider, git GitProvider) (ReleasePolicy, error) {
	if input.Mode != ModeRelease && input.Mode != ModeRerelease && input.Mode != ModeDocker && input.Mode != ModeGoreleaser {
		return ReleasePolicy{}, fmt.Errorf("invalid mode: %s", input.Mode)
	}

	if input.Mode == ModeDocker || input.Mode == ModeGoreleaser {
		return computeTagBasedPolicy(input, env, git)
	}

	if input.Publish != PublishTrue {
		return ReleasePolicy{
			ShouldPublish: false,
			Message:       "Info: No release markers found. Skipping tag creation and publish.",
		}, nil
	}

	resolvedTag := strings.TrimSpace(input.NextTag)
	if input.Mode == ModeRerelease && input.ResolveLatest && resolvedTag == "" {
		if git == nil {
			return ReleasePolicy{}, errors.New("git provider is required when resolve-latest-tag=true")
		}
		_ = exec.Command("git", "fetch", "--tags", "--force").Run()
		tag, err := git.GetLatestTag()
		if err != nil {
			return ReleasePolicy{}, err
		}
		resolvedTag = strings.TrimSpace(tag)
	}

	if resolvedTag == "" {
		return ReleasePolicy{}, fmt.Errorf("next-tag is required when publish=true")
	}

	if err := validateRequiredSecrets(input.RequiredSecrets, env); err != nil {
		return ReleasePolicy{}, err
	}

	version, err := parseTagVersion(resolvedTag)
	if err != nil {
		return ReleasePolicy{}, err
	}

	majorMinor, err := parseMajorMinor(version)
	if err != nil {
		return ReleasePolicy{}, err
	}

	dockerfile := detectDockerfileForWorkflow()
	shortSHA := shortenSHA(input.SHA)
	summary := buildSummary(input.Mode, input.LatestTag, resolvedTag, input.Image, version, shortSHA)

	return ReleasePolicy{
		ShouldPublish:    true,
		DockerVersion:    version,
		DockerMajorMinor: majorMinor,
		Dockerfile:       dockerfile,
		ReleaseTag:       resolvedTag,
		PublishMode:      "true",
		Message:          summary,
	}, nil
}

func computeTagBasedPolicy(input PolicyInput, env EnvProvider, git GitProvider) (ReleasePolicy, error) {
	resolvedTag := strings.TrimSpace(input.NextTag)

	if input.ResolveLatest && resolvedTag == "" {
		if git == nil {
			return ReleasePolicy{}, errors.New("git provider is required when resolve-latest-tag=true")
		}
		tag, err := git.GetLatestTag()
		if err != nil {
			return ReleasePolicy{}, err
		}
		resolvedTag = strings.TrimSpace(tag)
	}

	if resolvedTag == "" {
		return ReleasePolicy{}, fmt.Errorf("next-tag is required")
	}

	version, err := parseTagVersion(resolvedTag)
	if err != nil {
		return ReleasePolicy{}, err
	}

	majorMinor, err := parseMajorMinor(version)
	if err != nil {
		return ReleasePolicy{}, err
	}

	publishMode, err := resolvePublishMode(input.EventName, input.Publish, input.Mode)
	if err != nil {
		return ReleasePolicy{}, err
	}

	if publishMode == PublishTrue {
		if err := validateRequiredSecrets(input.RequiredSecrets, env); err != nil {
			return ReleasePolicy{}, err
		}
	}

	dockerfile := ""
	if input.Mode == ModeDocker {
		dockerfile = detectDockerfileForWorkflow()
	}

	msg := buildTagModeSummary(input.Mode, resolvedTag, publishMode)

	return ReleasePolicy{
		ShouldPublish:    publishMode == PublishTrue,
		DockerVersion:    version,
		DockerMajorMinor: majorMinor,
		Dockerfile:       dockerfile,
		ReleaseTag:       resolvedTag,
		PublishMode:      publishMode,
		Message:          msg,
	}, nil
}

func buildTagModeSummary(mode, tag, publishMode string) string {
	if mode == ModeGoreleaser {
		return fmt.Sprintf("Tag: %s\nMode: %s", tag, publishMode)
	}
	if mode == ModeDocker {
		return fmt.Sprintf("Source ref: %s", tag)
	}
	return ""
}

func buildSummary(mode, latestTag, nextTag, image, dockerVersion, shortSHA string) string {
	if mode == ModeRerelease {
		lines := []string{
			fmt.Sprintf("Re-released tag: %s", nextTag),
			"Docker image published.",
			"GoReleaser artifacts published.",
			"",
			"Image tags:",
			fmt.Sprintf("  - %s:%s", image, dockerVersion),
			fmt.Sprintf("  - %s:latest", image),
		}
		if shortSHA != "" {
			lines = append(lines, fmt.Sprintf("  - %s:sha-%s", image, shortSHA))
		}
		return strings.Join(lines, "\n")
	}

	prev := latestTag
	if strings.TrimSpace(prev) == "" {
		prev = "unknown"
	}

	lines := []string{
		fmt.Sprintf("Bumped from: %s", prev),
		fmt.Sprintf("Released new: %s", nextTag),
		"Docker image published.",
		"GoReleaser workflow will run from tag push.",
		"",
		"Image tags:",
		fmt.Sprintf("  - %s:%s", image, dockerVersion),
		fmt.Sprintf("  - %s:latest", image),
	}
	if shortSHA != "" {
		lines = append(lines, fmt.Sprintf("  - %s:sha-%s", image, shortSHA))
	}
	return strings.Join(lines, "\n")
}

func validateRequiredSecrets(required []string, env EnvProvider) error {
	missing := make([]string, 0)
	for _, key := range required {
		if env.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}

	if len(missing) == 0 {
		return nil
	}

	sort.Strings(missing)
	return fmt.Errorf("missing required secret(s): %s", strings.Join(missing, ", "))
}

func parseMajorMinor(version string) (string, error) {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid version format: %s", version)
	}
	return parts[0] + "." + parts[1], nil
}

func parseTagVersion(tag string) (string, error) {
	re := regexp.MustCompile(`^v([0-9]+\.[0-9]+\.[0-9]+)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(tag))
	if len(matches) != 2 {
		return "", fmt.Errorf("invalid tag format: %s", tag)
	}
	return matches[1], nil
}

func resolvePublishMode(eventName, publishInput, mode string) (string, error) {
	if eventName == "push" {
		return PublishTrue, nil
	}

	p := strings.TrimSpace(publishInput)
	if p == "" {
		if mode == ModeDocker {
			return PublishTrue, nil
		}
		return PublishFalse, nil
	}

	if p != PublishTrue && p != PublishFalse {
		return "", fmt.Errorf("invalid publish mode: %s", p)
	}

	return p, nil
}
