package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// parseOutputs is a helper to parse GITHUB_OUTPUT format into a map
func parseOutputs(output string) map[string]string {
	outputs := make(map[string]string)
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Handle multiline outputs (summary_message<<EOF format)
		if strings.Contains(line, "<<EOF") {
			parts := strings.SplitN(line, "<<", 2)
			if len(parts) == 2 {
				key := parts[0]
				// Skip until EOF
				for i++; i < len(lines); i++ {
					if strings.TrimSpace(lines[i]) == "EOF" {
						break
					}
				}
				outputs[key] = "[multiline]"
				continue
			}
		}

		// Normal key=value format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			outputs[parts[0]] = parts[1]
		}
	}

	return outputs
}

// setupPlanTest creates a temp output file and sets environment
func setupPlanTest(t *testing.T) (string, func()) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "github_output.txt")

	// Change to temp directory to isolate plan.json
	origDir, _ := os.Getwd()
	os.Chdir(tempDir)

	os.Setenv("GITHUB_OUTPUT", outputFile)
	os.Setenv("GITHUB_EVENT_NAME", "push")

	cleanup := func() {
		os.Chdir(origDir)
		os.Unsetenv("GITHUB_OUTPUT")
		os.Unsetenv("GITHUB_EVENT_NAME")
	}

	return outputFile, cleanup
}

// verifyRequiredOutputs checks that all critical outputs are present
func verifyRequiredOutputs(t *testing.T, outputs map[string]string) {
	required := []string{"mode", "tag_latest", "tag_next", "tag_release", "release_skip"}
	for _, key := range required {
		if _, ok := outputs[key]; !ok {
			t.Errorf("MISSING CRITICAL OUTPUT: %s", key)
		}
	}
}

// TestPlanOutputPatchBump tests plan output with explicit patch bump
func TestPlanOutputPatchBump(t *testing.T) {
	outputFile, cleanup := setupPlanTest(t)
	defer cleanup()

	// Override event to workflow_dispatch so bump parameter is used
	os.Setenv("GITHUB_EVENT_NAME", "workflow_dispatch")

	// Create a plan with mock git
	plan := &Plan{
		Mode:                "release",
		Bump:                "patch",
		DryRun:              true,
		UseDocker:           true,
		UseGoreleaser:       true,
		UseGoreleaserDocker: true,
		DockerImage:         "karloie/kompass",
	}

	// Mock git to return v1.2.2 as latest tag
	mockGit := &GitProviderMock{
		LatestTag: "v1.2.2",
	}

	err := runPlanClean(plan, mockGit, nil)
	if err != nil {
		t.Fatalf("runPlan failed: %v", err)
	}

	outputBytes, _ := os.ReadFile(outputFile)
	outputs := parseOutputs(string(outputBytes))

	t.Logf("=== PATCH BUMP OUTPUTS ===")
	for k, v := range outputs {
		t.Logf("%s=%s", k, v)
	}

	verifyRequiredOutputs(t, outputs)

	if outputs["release_skip"] != "false" {
		t.Errorf("Expected release_skip=false with -bump=patch, got: %s", outputs["release_skip"])
	}

	if outputs["mode"] != "release" {
		t.Errorf("Expected mode=release, got: %s", outputs["mode"])
	}

	if outputs["tag_latest"] != "v1.2.2" {
		t.Errorf("Expected tag_latest=v1.2.2, got: %s", outputs["tag_latest"])
	}

	if outputs["tag_next"] != "v1.2.3" {
		t.Errorf("Expected tag_next=v1.2.3 (patch bump from v1.2.2), got: %s", outputs["tag_next"])
	}

	if outputs["version_clean"] != "1.2.3" {
		t.Errorf("Expected version_clean=1.2.3, got: %s", outputs["version_clean"])
	}
}

// TestPlanOutputMinorBump tests plan output with minor bump
func TestPlanOutputMinorBump(t *testing.T) {
	outputFile, cleanup := setupPlanTest(t)
	defer cleanup()

	err := runPlan([]string{"-mode=release", "-bump=minor"})
	if err != nil {
		t.Fatalf("runPlan failed: %v", err)
	}

	outputBytes, _ := os.ReadFile(outputFile)
	outputs := parseOutputs(string(outputBytes))

	t.Logf("=== MINOR BUMP OUTPUTS ===")
	for k, v := range outputs {
		t.Logf("%s=%s", k, v)
	}

	verifyRequiredOutputs(t, outputs)

	if outputs["release_skip"] != "false" {
		t.Errorf("Expected release_skip=false with -bump=minor, got: %s", outputs["release_skip"])
	}
}

// TestPlanOutputMajorBump tests plan output with major bump
func TestPlanOutputMajorBump(t *testing.T) {
	outputFile, cleanup := setupPlanTest(t)
	defer cleanup()

	err := runPlan([]string{"-mode=release", "-bump=major"})
	if err != nil {
		t.Fatalf("runPlan failed: %v", err)
	}

	outputBytes, _ := os.ReadFile(outputFile)
	outputs := parseOutputs(string(outputBytes))

	t.Logf("=== MAJOR BUMP OUTPUTS ===")
	for k, v := range outputs {
		t.Logf("%s=%s", k, v)
	}

	verifyRequiredOutputs(t, outputs)

	if outputs["release_skip"] != "false" {
		t.Errorf("Expected release_skip=false with -bump=major, got: %s", outputs["release_skip"])
	}
}

// TestPlanOutputReleaseNoBump tests release mode without explicit bump (depends on commits)
func TestPlanOutputReleaseNoBump(t *testing.T) {
	outputFile, cleanup := setupPlanTest(t)
	defer cleanup()

	err := runPlan([]string{"-mode=release"})
	if err != nil {
		t.Fatalf("runPlan failed: %v", err)
	}

	outputBytes, _ := os.ReadFile(outputFile)
	outputs := parseOutputs(string(outputBytes))

	t.Logf("=== RELEASE NO BUMP OUTPUTS ===")
	for k, v := range outputs {
		t.Logf("%s=%s", k, v)
	}

	verifyRequiredOutputs(t, outputs)

	// release_skip depends on whether repo has conventional commits
	// Just verify it exists and is either "true" or "false"
	if skip := outputs["release_skip"]; skip != "true" && skip != "false" {
		t.Errorf("release_skip must be 'true' or 'false', got: %s", skip)
	}
}

// TestPlanOutputDockerMode tests docker mode
func TestPlanOutputDockerMode(t *testing.T) {
	outputFile, cleanup := setupPlanTest(t)
	defer cleanup()

	err := runPlan([]string{"-mode=docker", "-next-tag=v1.2.3", "-latest-tag=v1.2.2"})
	if err != nil {
		t.Fatalf("runPlan failed: %v", err)
	}

	outputBytes, _ := os.ReadFile(outputFile)
	outputs := parseOutputs(string(outputBytes))

	t.Logf("=== DOCKER MODE OUTPUTS ===")
	for k, v := range outputs {
		t.Logf("%s=%s", k, v)
	}

	verifyRequiredOutputs(t, outputs)

	if outputs["mode"] != "docker" {
		t.Errorf("Expected mode=docker, got: %s", outputs["mode"])
	}

	if outputs["tag_next"] != "v1.2.3" {
		t.Errorf("Expected tag_next=v1.2.3, got: %s", outputs["tag_next"])
	}

	// Docker mode WITHOUT a Dockerfile should skip
	if outputs["release_skip"] != "true" {
		t.Errorf("Expected release_skip=true in docker mode when no Dockerfile exists, got: %s", outputs["release_skip"])
	}
}

// TestPlanOutputRereleaseMode tests rerelease mode
func TestPlanOutputRereleaseMode(t *testing.T) {
	outputFile, cleanup := setupPlanTest(t)
	defer cleanup()

	err := runPlan([]string{"-mode=rerelease"})
	if err != nil {
		t.Fatalf("runPlan failed: %v", err)
	}

	outputBytes, _ := os.ReadFile(outputFile)
	outputs := parseOutputs(string(outputBytes))

	t.Logf("=== RERELEASE MODE OUTPUTS ===")
	for k, v := range outputs {
		t.Logf("%s=%s", k, v)
	}

	verifyRequiredOutputs(t, outputs)

	if outputs["mode"] != "rerelease" {
		t.Errorf("Expected mode=rerelease, got: %s", outputs["mode"])
	}

	// Rerelease should never skip
	if outputs["release_skip"] != "false" {
		t.Errorf("Expected release_skip=false in rerelease mode, got: %s", outputs["release_skip"])
	}

	// tag_latest should contain the actual latest tag from git
	if outputs["tag_latest"] == "" {
		t.Errorf("Expected tag_latest to have a value in rerelease mode, got empty string")
	}
}

// TestPlanOutputGoreleaserMode tests goreleaser mode
func TestPlanOutputGoreleaserMode(t *testing.T) {
	outputFile, cleanup := setupPlanTest(t)
	defer cleanup()

	err := runPlan([]string{"-mode=goreleaser"})
	if err != nil {
		t.Fatalf("runPlan failed: %v", err)
	}

	outputBytes, _ := os.ReadFile(outputFile)
	outputs := parseOutputs(string(outputBytes))

	t.Logf("=== GORELEASER MODE OUTPUTS ===")
	for k, v := range outputs {
		t.Logf("%s=%s", k, v)
	}

	verifyRequiredOutputs(t, outputs)

	if outputs["mode"] != "goreleaser" {
		t.Errorf("Expected mode=goreleaser, got: %s", outputs["mode"])
	}
}

// TestPlanOutputWithDryRun tests that dry-run doesn't affect outputs
func TestPlanOutputWithDryRun(t *testing.T) {
	outputFile, cleanup := setupPlanTest(t)
	defer cleanup()

	err := runPlan([]string{"-mode=release", "-bump=patch", "-dry-run=true"})
	if err != nil {
		t.Fatalf("runPlan failed: %v", err)
	}

	outputBytes, _ := os.ReadFile(outputFile)
	outputs := parseOutputs(string(outputBytes))

	t.Logf("=== DRY RUN OUTPUTS ===")
	for k, v := range outputs {
		t.Logf("%s=%s", k, v)
	}

	verifyRequiredOutputs(t, outputs)

	// Dry run still computes everything
	if outputs["release_skip"] != "false" {
		t.Errorf("Expected release_skip=false even in dry-run, got: %s", outputs["release_skip"])
	}
}

// TestPlanOutputCustomImage tests custom docker image parameter
func TestPlanOutputCustomImage(t *testing.T) {
	outputFile, cleanup := setupPlanTest(t)
	defer cleanup()

	err := runPlan([]string{"-mode=release", "-bump=patch", "-image=myorg/myapp"})
	if err != nil {
		t.Fatalf("runPlan failed: %v", err)
	}

	outputBytes, _ := os.ReadFile(outputFile)
	outputs := parseOutputs(string(outputBytes))

	t.Logf("=== CUSTOM IMAGE OUTPUTS ===")
	for k, v := range outputs {
		t.Logf("%s=%s", k, v)
	}

	if outputs["docker_image"] != "myorg/myapp" {
		t.Errorf("Expected docker_image=myorg/myapp, got: %s", outputs["docker_image"])
	}
}

// TestPlanOutputConsistency verifies that all outputs are written in all modes
func TestPlanOutputConsistency(t *testing.T) {
	modes := []string{"release", "docker", "rerelease", "goreleaser"}

	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			outputFile, cleanup := setupPlanTest(t)
			defer cleanup()

			args := []string{"-mode=" + mode}
			if mode == "docker" {
				args = append(args, "-next-tag=v1.0.0")
			}
			if mode == "release" || mode == "goreleaser" {
				args = append(args, "-bump=patch")
			}

			err := runPlan(args)
			if err != nil {
				t.Fatalf("runPlan failed for mode %s: %v", mode, err)
			}

			outputBytes, _ := os.ReadFile(outputFile)
			outputs := parseOutputs(string(outputBytes))

			verifyRequiredOutputs(t, outputs)

			// Verify mode is set correctly
			if outputs["mode"] != mode {
				t.Errorf("Expected mode=%s, got: %s", mode, outputs["mode"])
			}
		})
	}
}
