package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func runVersion(args []string) error {
	fs := newFlagSet("version")
	bump := fs.String("bump", "", "patch|minor|major (optional, auto-detect from commits if empty)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	eventName := os.Getenv(EnvGitHubEventName)
	githubOutput := os.Getenv(EnvGitHubOutput)
	token := os.Getenv(EnvGitHubToken)

	git := &GitProviderReal{}
	pr := &PRProviderReal{token: token}

	latest, next, publish, err := computeVersion(eventName, *bump, git, pr)
	if err != nil {
		return err
	}

	if publish == PublishSkip {
		fmt.Println("⏭️  No release markers found. Skipping release.")
		writeOutput(githubOutput, "bump", PublishSkip)
		return nil
	}

	fmt.Printf("🔄 Bumped from: %s\n", latest)
	fmt.Printf("🎉 Released new: %s\n", next)
	writeOutput(githubOutput, OutputTagLatest, latest)
	writeOutput(githubOutput, OutputTagNext, next)
	writeOutput(githubOutput, OutputPublish, PublishTrue)
	return nil
}

func computeVersion(eventName, bumpInput string, git GitProvider, pr PRProvider) (latest, next, publish string, err error) {
	latest, err = git.GetLatestTag()
	if err != nil {
		latest = "v0.0.0"
	}

	version := strings.TrimPrefix(latest, "v")
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid tag format: %s", latest)
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])

	bump := ""
	if eventName == "push" {
		if pr != nil {
			labels, _ := pr.GetMergedPRLabels()
			if labels != "" {
				bump = parsePRLabels(labels)
			}
		}

		if bump == "" {
			bump, err = analyzeCommits(latest, git)
			if err != nil {
				return "", "", "", err
			}
		}

		if bump == "" {
			return latest, "", PublishSkip, nil
		}
	} else {
		if bumpInput == "" {
			return "", "", "", fmt.Errorf("manual release requires -bump flag (patch|minor|major)")
		}
		bump = bumpInput
	}

	switch bump {
	case BumpMajor:
		major++
		minor = 0
		patch = 0
	case BumpMinor:
		minor++
		patch = 0
	case BumpPatch:
		patch++
	default:
		return "", "", "", fmt.Errorf("invalid bump: %s", bump)
	}

	next = fmt.Sprintf("v%d.%d.%d", major, minor, patch)

	exists, err := git.TagExists(next)
	if err != nil {
		return "", "", "", err
	}
	if exists {
		return "", "", "", fmt.Errorf("tag %s already exists", next)
	}

	return latest, next, PublishTrue, nil
}

func parsePRLabels(labels string) string {
	if strings.Contains(labels, PRLabelReleaseMajor) {
		return BumpMajor
	}
	if strings.Contains(labels, PRLabelReleaseMinor) {
		return BumpMinor
	}
	if strings.Contains(labels, PRLabelReleasePatch) {
		return BumpPatch
	}
	return ""
}

func analyzeCommits(latestTag string, git GitProvider) (string, error) {
	log, err := git.GetCommitLog(latestTag)
	if err != nil {
		return "", err
	}

	if regexp.MustCompile(`BREAKING CHANGE|feat!`).MatchString(log) {
		return BumpMajor, nil
	}
	if regexp.MustCompile(`^[a-f0-9]+ feat`).MatchString(log) {
		return BumpMinor, nil
	}
	if regexp.MustCompile(`^[a-f0-9]+ fix`).MatchString(log) {
		return BumpPatch, nil
	}
	return "", nil
}
