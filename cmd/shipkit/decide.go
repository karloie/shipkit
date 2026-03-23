package main

import (
	"fmt"
	"os"
	"strings"
)

// DecideInputs contains all inputs for the decide command
type DecideInputs struct {
	// Workflow inputs
	Mode   string
	DryRun bool

	// Job results (success or skipped = ok)
	Build string // Single build result from `shipkit build`
	Tag   string
}

// DecideOutputs contains the decision outputs
type DecideOutputs struct {
	ShouldPublish bool
}

// Decide makes publish decision based on build success
func Decide(inputs DecideInputs) DecideOutputs {
	outputs := DecideOutputs{}

	// Build & tag must pass (or be skipped)
	buildPassed := jobOk(inputs.Build)
	tagPassed := jobOk(inputs.Tag)
	allOk := buildPassed && tagPassed && !inputs.DryRun

	// Single publish decision: everything passed
	outputs.ShouldPublish = allOk

	return outputs
}

// jobOk returns true if job result is success or skipped
func jobOk(result string) bool {
	result = strings.ToLower(strings.TrimSpace(result))
	return result == "success" || result == "skipped"
}

func runDecide(args []string) error {
	fs := newFlagSet("decide")

	mode := fs.String("mode", DefaultMode, "Release mode")
	dryRun := fs.Bool("dry-run", false, "Dry run")
	build := fs.String("result-build", "skipped", "Build result")
	tag := fs.String("result-tag", "skipped", "Tag result")
	parseFlagsOrExit(fs, args)

	inputs := DecideInputs{
		Mode:   strings.TrimSpace(*mode),
		DryRun: *dryRun,
		Build:  strings.TrimSpace(*build),
		Tag:    strings.TrimSpace(*tag),
	}

	// Log inputs
	logInputs(map[string]string{
		"mode":         inputs.Mode,
		"dry-run":      fmt.Sprintf("%v", inputs.DryRun),
		"result-build": inputs.Build,
		"result-tag":   inputs.Tag,
	})

	fmt.Println("::group::Decide publish actions")
	defer fmt.Println("::endgroup::")

	outputs := Decide(inputs)

	// Write outputs
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput == "" {
		return fmt.Errorf("GITHUB_OUTPUT environment variable not set")
	}

	writeBoolOutput(githubOutput, "should_publish", outputs.ShouldPublish)

	// Log outputs
	logOutputs(map[string]string{
		"should_publish": fmt.Sprintf("%v", outputs.ShouldPublish),
	})

	return nil
}
