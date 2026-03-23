package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPlanOutput is a table-driven test for all plan output variations
func TestPlanOutput(t *testing.T) {
	tests := []planTestCase{
		{
			name:      "patch_bump",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"mode":          "release",
				"tag_latest":    "v1.2.2",
				"tag_next":      "v1.2.3",
				"version_clean": "1.2.3",
				"release_skip":  "false",
			},
		},
		{
			name:      "initial_release_no_tags",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "", // No tags in repository
			expectation: map[string]string{
				"mode":                "release",
				"tag_latest":          "v0.0.0", // Baseline when no tags exist
				"tag_next":            "v0.0.1", // First release starts at v0.0.1
				"version_clean":       "0.0.1",
				"version_major_minor": "", // 0.x versions return empty
				"release_skip":        "false",
			},
		},
		{
			name:      "minor_bump",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "minor",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"mode":         "release",
				"tag_next":     "v1.3.0",
				"release_skip": "false",
			},
		},
		{
			name:      "major_bump",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "major",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"mode":         "release",
				"tag_next":     "v2.0.0",
				"release_skip": "false",
			},
		},
		{
			name:      "release_no_bump",
			eventName: "push",
			plan: &Plan{
				Mode:                "release",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			validateFunc: func(t *testing.T, outputs map[string]string) {
				if skip := outputs["release_skip"]; skip != "true" && skip != "false" {
					t.Errorf("release_skip must be 'true' or 'false', got: %s", skip)
				}
			},
		},
		{
			name:      "rerelease_mode",
			eventName: "push",
			plan: &Plan{
				Mode:                "rerelease",
				ResolveLatestTag:    true,
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"mode":         "rerelease",
				"release_skip": "false",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				if outputs["tag_release"] == "" {
					t.Errorf("Expected tag_release to have a value in rerelease mode, got empty string")
				}
			},
		},
		{
			name:      "goreleaser_mode",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "goreleaser",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"mode": "goreleaser",
			},
		},
		{
			name:      "dry_run",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"release_skip": "false",
			},
		},
		{
			name:      "version_0x_major_minor_empty",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "minor",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v0.1.5",
			expectation: map[string]string{
				"mode":                "release",
				"tag_latest":          "v0.1.5",
				"tag_next":            "v0.2.0",
				"version_clean":       "0.2.0",
				"version_major_minor": "", // 0.x versions should return empty
			},
		},
		{
			name:      "rerelease_docker_tag_latest_false",
			eventName: "push",
			plan: &Plan{
				Mode:                "rerelease",
				ResolveLatestTag:    true,
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"mode":              "rerelease",
				"docker_tag_latest": "false", // Rerelease should not tag as latest
			},
		},
	}

	runPlanTests(t, tests)
}

// planTestCase defines the structure for table-driven plan tests
type planTestCase struct {
	name         string
	eventName    string
	plan         *Plan
	mockLatest   string
	setupFunc    func(*testing.T)
	expectation  map[string]string
	validateFunc func(*testing.T, map[string]string)
}

// runPlanTests executes a slice of plan test cases
func runPlanTests(t *testing.T, tests []planTestCase) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputFile, cleanup := setupPlanTest(t)
			defer cleanup()

			// Set event name if specified
			if tt.eventName != "" {
				os.Setenv("GITHUB_EVENT_NAME", tt.eventName)
			}

			// Run setup function if provided (e.g., create test files)
			if tt.setupFunc != nil {
				tt.setupFunc(t)
			}

			mockGit := &GitProviderMock{
				LatestTag: tt.mockLatest,
			}

			err := runPlanClean(tt.plan, mockGit, nil)
			if err != nil {
				t.Fatalf("runPlan failed: %v", err)
			}

			outputBytes, _ := os.ReadFile(outputFile)
			outputs := parseOutputs(string(outputBytes))

			// Log outputs for debugging
			t.Logf("=== %s OUTPUTS ===", strings.ToUpper(tt.name))
			for k, v := range outputs {
				t.Logf("%s=%s", k, v)
			}

			// Verify required outputs
			verifyRequiredOutputs(t, outputs)

			// Check expected values
			for key, expectedValue := range tt.expectation {
				if actualValue := outputs[key]; actualValue != expectedValue {
					t.Errorf("Expected %s=%s, got: %s", key, expectedValue, actualValue)
				}
			}

			// Run custom validation if provided
			if tt.validateFunc != nil {
				tt.validateFunc(t, outputs)
			}
		})
	}
}

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
