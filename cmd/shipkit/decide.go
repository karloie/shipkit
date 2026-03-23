package main

import (
	"fmt"
	"os"
	"strings"
)

// DecideInputs contains all inputs for the decide command
type DecideInputs struct {
	// Plan outputs (what should run)
	ShouldRunNpmBuild    bool
	ShouldRunGoBuild     bool
	ShouldRunMavenBuild  bool
	ShouldRunDockerBuild bool

	// Workflow inputs
	Mode             string
	DryRun           bool
	UseGoreleaser    bool
	GoreleaserDocker bool

	// Job results (success or skipped = ok)
	NpmBuild       string
	GoBuild        string
	MavenBuild     string
	DockerBuild    string
	Tag            string
	UpdateVersions string
}

// DecideOutputs contains the decision outputs
type DecideOutputs struct {
	ShouldPublishNpm    bool
	ShouldPublishMaven  bool
	ShouldPublishDocker bool
	ShouldPublishGo     bool
}

// Decide makes publish decisions based on what should run and what succeeded
func Decide(inputs DecideInputs) DecideOutputs {
	outputs := DecideOutputs{}

	// All builds & setup must pass (or be skipped if not run)
	buildsPassed := jobOk(inputs.NpmBuild) && jobOk(inputs.GoBuild) &&
		jobOk(inputs.MavenBuild) && jobOk(inputs.DockerBuild)
	setupPassed := jobOk(inputs.Tag) && jobOk(inputs.UpdateVersions)
	allOk := buildsPassed && setupPassed && !inputs.DryRun

	// npm publish: should have run npm-build AND everything passed
	outputs.ShouldPublishNpm = inputs.ShouldRunNpmBuild && allOk

	// Maven publish: should have run maven-build AND everything passed
	outputs.ShouldPublishMaven = inputs.ShouldRunMavenBuild && allOk

	// Docker publish: should have run docker-build AND everything passed AND goreleaser doesn't handle it AND not goreleaser mode
	outputs.ShouldPublishDocker = inputs.ShouldRunDockerBuild && allOk &&
		!inputs.GoreleaserDocker && inputs.Mode != ModeGoreleaser

	// Go publish: use_goreleaser AND everything passed
	outputs.ShouldPublishGo = inputs.UseGoreleaser && allOk

	return outputs
}

// jobOk returns true if job result is success or skipped
func jobOk(result string) bool {
	result = strings.ToLower(strings.TrimSpace(result))
	return result == "success" || result == "skipped"
}

func runDecide(args []string) error {
	fs := newFlagSet("decide")

	// Workflow inputs
	mode := fs.String("mode", DefaultMode, "policy mode: release|rerelease|docker|goreleaser")
	dryRun := fs.Bool("dry-run", false, "dry run mode")
	useGoreleaser := fs.Bool("use-goreleaser", true, "use goreleaser")

	// Plan outputs (what should run)
	shouldRunNpmBuild := fs.Bool("should-run-npm-build", false, "should run npm-build from plan")
	shouldRunGoBuild := fs.Bool("should-run-go-build", false, "should run go-build from plan")
	shouldRunMavenBuild := fs.Bool("should-run-maven-build", false, "should run maven-build from plan")
	shouldRunDockerBuild := fs.Bool("should-run-docker-build", false, "should run docker-build from plan")
	goreleaserDocker := fs.Bool("goreleaser-docker", false, "goreleaser handles docker from plan")

	// Job results
	npmBuild := fs.String("npm-build-result", "skipped", "npm-build job result")
	goBuild := fs.String("go-build-result", "skipped", "go-build job result")
	mavenBuild := fs.String("maven-build-result", "skipped", "maven-build job result")
	dockerBuild := fs.String("docker-build-result", "skipped", "docker-build job result")
	tag := fs.String("tag-result", "skipped", "tag job result")
	updateVersions := fs.String("update-versions-result", "skipped", "update-versions job result")

	if err := fs.Parse(args); err != nil {
		return err
	}

	inputs := DecideInputs{
		Mode:                 strings.TrimSpace(*mode),
		DryRun:               *dryRun,
		UseGoreleaser:        *useGoreleaser,
		GoreleaserDocker:     *goreleaserDocker,
		ShouldRunNpmBuild:    *shouldRunNpmBuild,
		ShouldRunGoBuild:     *shouldRunGoBuild,
		ShouldRunMavenBuild:  *shouldRunMavenBuild,
		ShouldRunDockerBuild: *shouldRunDockerBuild,
		NpmBuild:             strings.TrimSpace(*npmBuild),
		GoBuild:              strings.TrimSpace(*goBuild),
		MavenBuild:           strings.TrimSpace(*mavenBuild),
		DockerBuild:          strings.TrimSpace(*dockerBuild),
		Tag:                  strings.TrimSpace(*tag),
		UpdateVersions:       strings.TrimSpace(*updateVersions),
	}

	// Log inputs
	logInputs(map[string]string{
		"mode":                    inputs.Mode,
		"dry-run":                 fmt.Sprintf("%v", inputs.DryRun),
		"use-goreleaser":          fmt.Sprintf("%v", inputs.UseGoreleaser),
		"goreleaser-docker":       fmt.Sprintf("%v", inputs.GoreleaserDocker),
		"should-run-npm-build":    fmt.Sprintf("%v", inputs.ShouldRunNpmBuild),
		"should-run-go-build":     fmt.Sprintf("%v", inputs.ShouldRunGoBuild),
		"should-run-maven-build":  fmt.Sprintf("%v", inputs.ShouldRunMavenBuild),
		"should-run-docker-build": fmt.Sprintf("%v", inputs.ShouldRunDockerBuild),
		"npm-build-result":        inputs.NpmBuild,
		"go-build-result":         inputs.GoBuild,
		"maven-build-result":      inputs.MavenBuild,
		"docker-build-result":     inputs.DockerBuild,
		"tag-result":              inputs.Tag,
		"update-versions-result":  inputs.UpdateVersions,
	})

	fmt.Println("::group::Decide publish actions")

	outputs := Decide(inputs)

	fmt.Printf("  📦 npm publish: %v\n", outputs.ShouldPublishNpm)
	fmt.Printf("  ☕ maven publish: %v\n", outputs.ShouldPublishMaven)
	fmt.Printf("  🐳 docker publish: %v\n", outputs.ShouldPublishDocker)
	fmt.Printf("  🚀 go publish: %v\n", outputs.ShouldPublishGo)

	fmt.Println("::endgroup::")

	// Write outputs
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput == "" {
		return fmt.Errorf("GITHUB_OUTPUT environment variable not set")
	}

	writeBoolOutput(githubOutput, "should_publish_npm", outputs.ShouldPublishNpm)
	writeBoolOutput(githubOutput, "should_publish_maven", outputs.ShouldPublishMaven)
	writeBoolOutput(githubOutput, "should_publish_docker", outputs.ShouldPublishDocker)
	writeBoolOutput(githubOutput, "should_publish_go", outputs.ShouldPublishGo)

	return nil
}
