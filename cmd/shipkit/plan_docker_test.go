package main

import (
	"os"
	"testing"
)

// TestPlanDocker is a table-driven test for docker-related plan functionality
func TestPlanDocker(t *testing.T) {
	tests := []planTestCase{
		{
			name:      "docker_mode_no_dockerfile",
			eventName: "push",
			plan: &Plan{
				Mode:                "docker",
				TagNext:             "v1.2.3",
				TagLatest:           "v1.2.2",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"mode":         "docker",
				"tag_next":     "v1.2.3",
				"release_skip": "true", // No Dockerfile in temp dir
				"docker_file":  "",     // No Dockerfile detected
				"has_docker":   "false",
			},
		},
		{
			name:      "docker_mode_with_dockerfile",
			eventName: "push",
			plan: &Plan{
				Mode:                "docker",
				TagNext:             "v1.2.3",
				TagLatest:           "v1.2.2",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				if err := os.WriteFile("Dockerfile", []byte("FROM scratch"), 0644); err != nil {
					t.Fatalf("failed to create Dockerfile: %v", err)
				}
			},
			expectation: map[string]string{
				"mode":                  "docker",
				"tag_next":              "v1.2.3",
				"goreleaser_dockerfile": "Dockerfile", // Dockerfile detected
				"release_skip":          "false",      // Dockerfile exists, so don't skip
			},
		},
		{
			name:      "docker_mode_with_containerfile",
			eventName: "push",
			plan: &Plan{
				Mode:                "docker",
				TagNext:             "v1.2.3",
				TagLatest:           "v1.2.2",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				if err := os.WriteFile("Containerfile", []byte("FROM scratch"), 0644); err != nil {
					t.Fatalf("failed to create Containerfile: %v", err)
				}
			},
			expectation: map[string]string{
				"mode":                  "docker",
				"tag_next":              "v1.2.3",
				"goreleaser_dockerfile": "Containerfile", // Containerfile detected
				"release_skip":          "false",         // Containerfile exists, so don't skip
			},
		},
		{
			name:      "custom_image",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "myorg/myapp",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"docker_image": "myorg/myapp",
			},
		},
		{
			name:      "release_with_dockerfile",
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
				if err := os.WriteFile("Dockerfile", []byte("FROM scratch"), 0644); err != nil {
					t.Fatalf("failed to create Dockerfile: %v", err)
				}
			},
			expectation: map[string]string{
				"mode":                  "release",
				"tag_next":              "v1.2.3",
				"goreleaser_dockerfile": "Dockerfile",
				"has_docker":            "true",
			},
		},
		{
			name:      "rerelease_with_dockerfile",
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
			setupFunc: func(t *testing.T) {
				if err := os.WriteFile("Dockerfile", []byte("FROM scratch"), 0644); err != nil {
					t.Fatalf("failed to create Dockerfile: %v", err)
				}
			},
			expectation: map[string]string{
				"mode":                  "rerelease",
				"docker_tag_latest":     "false",
				"goreleaser_dockerfile": "Dockerfile",
				"has_docker":            "true",
			},
		},
	}

	runPlanTests(t, tests)
}
