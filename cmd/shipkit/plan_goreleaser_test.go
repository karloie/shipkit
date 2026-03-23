package main

import (
	"os"
	"testing"
)

// TestPlanGoreleaser is a table-driven test for goreleaser-related plan functionality
func TestPlanGoreleaser(t *testing.T) {
	tests := []planTestCase{
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
			name:      "custom_goreleaser_yml",
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
			setupFunc: func(t *testing.T) {
				if err := os.WriteFile(".goreleaser.yml", []byte("project_name: test"), 0644); err != nil {
					t.Fatalf("failed to create .goreleaser.yml: %v", err)
				}
			},
			expectation: map[string]string{
				"goreleaser_config": ".goreleaser.yml",
			},
		},
		{
			name:      "custom_goreleaser_yaml",
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
			setupFunc: func(t *testing.T) {
				if err := os.WriteFile(".goreleaser.yaml", []byte("project_name: test"), 0644); err != nil {
					t.Fatalf("failed to create .goreleaser.yaml: %v", err)
				}
			},
			expectation: map[string]string{
				"goreleaser_config": ".goreleaser.yaml",
			},
		},
	}

	runPlanTests(t, tests)
}
