package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// DecideOutputs contains the decision outputs
type DecideOutputs struct {
	ShouldRelease bool
}

// Decide makes publish decision based on verification and publish readiness.
func Decide(plan *Plan) DecideOutputs {
	outputs := DecideOutputs{}

	// Initialize JobResults map if nil
	if plan.JobResults == nil {
		plan.JobResults = make(map[string]string)
	}

	// Verification steps and tag must pass (or be skipped).
	buildPassed := jobOk(plan.JobResults["build"])
	testPassed := jobOk(plan.JobResults["test"])
	integrationTestPassed := jobOk(plan.JobResults["integration-test"])
	tagPassed := jobOk(plan.JobResults["tag"])
	allOk := buildPassed && testPassed && integrationTestPassed && tagPassed && !plan.DryRun

	// Single publish decision: everything passed
	outputs.ShouldRelease = allOk

	return outputs
}

// jobOk returns true if job result is success or skipped
func jobOk(result string) bool {
	result = strings.ToLower(strings.TrimSpace(result))
	return result == "success" || result == "skipped"
}

func runDecide(args []string) error {
	// Log raw args BEFORE parsing
	logInputs(map[string]string{
		"raw_args": strings.Join(args, " "),
	})

	// Load plan.json if it exists
	var plan Plan
	data, err := os.ReadFile("plan.json")
	if err == nil {
		if err := json.Unmarshal(data, &plan); err != nil {
			return fmt.Errorf("failed to parse plan.json: %w", err)
		}
	}

	// Initialize JobResults map if needed
	if plan.JobResults == nil {
		plan.JobResults = make(map[string]string)
	}

	fs := newFlagSet("decide")

	mode := fs.String("mode", plan.Mode, "Release mode")
	dryRun := fs.Bool("dry-run", plan.DryRun, "Dry run")
	build := fs.String("result-build", "skipped", "Build result")
	test := fs.String("result-test", "skipped", "Test result")
	integrationTest := fs.String("result-integration-test", "skipped", "Integration test result")
	tag := fs.String("result-tag", "skipped", "Tag result")
	parseFlagsOrExit(fs, args)

	// Override plan with CLI args
	if *mode != "" {
		plan.Mode = strings.TrimSpace(*mode)
	}
	plan.DryRun = *dryRun
	plan.JobResults["build"] = strings.TrimSpace(*build)
	plan.JobResults["test"] = strings.TrimSpace(*test)
	plan.JobResults["integration-test"] = strings.TrimSpace(*integrationTest)
	plan.JobResults["tag"] = strings.TrimSpace(*tag)

	// Log inputs
	logInputs(map[string]string{
		"mode":         plan.Mode,
		"dry-run":      fmt.Sprintf("%v", plan.DryRun),
		"result-build": plan.JobResults["build"],
		"result-test":  plan.JobResults["test"],
		"result-int":   plan.JobResults["integration-test"],
		"result-tag":   plan.JobResults["tag"],
	})

	fmt.Println("::group::Decide release actions")
	defer fmt.Println("::endgroup::")

	outputs := Decide(&plan)

	// Write outputs
	githubOutput := os.Getenv("GITHUB_OUTPUT")
	if githubOutput == "" {
		return fmt.Errorf("GITHUB_OUTPUT environment variable not set")
	}

	writeBoolOutput(githubOutput, "should_release", outputs.ShouldRelease)

	// Log outputs
	logOutputs(map[string]string{
		"should_release": fmt.Sprintf("%v", outputs.ShouldRelease),
	})

	return nil
}
