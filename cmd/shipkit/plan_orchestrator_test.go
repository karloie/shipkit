package main

import (
	"os"
	"testing"
)

// TestPlanOrchestrator tests build orchestrator detection in plan output
func TestPlanOrchestrator(t *testing.T) {
	tests := []planTestCase{
		{
			name:      "makefile",
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
				if err := os.WriteFile("Makefile", []byte("all:\n\techo hello"), 0644); err != nil {
					t.Fatalf("failed to create Makefile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "make",
				"has_makefile":       "true",
			},
		},
		{
			name:      "justfile",
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
				if err := os.WriteFile("justfile", []byte("default:\n  echo hello"), 0644); err != nil {
					t.Fatalf("failed to create justfile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "just",
				"has_justfile":       "true",
			},
		},
		{
			name:      "taskfile",
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
				if err := os.WriteFile("Taskfile.yml", []byte("version: '3'\ntasks:\n  default:\n    cmds:\n      - echo hello"), 0644); err != nil {
					t.Fatalf("failed to create Taskfile.yml: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "task",
				"has_taskfile":       "true",
			},
		},
		{
			name:      "convention_no_orchestrator",
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
				"build_orchestrator": "convention",
				"has_makefile":       "false",
				"has_justfile":       "false",
				"has_taskfile":       "false",
			},
		},
	}

	runPlanTests(t, tests)
}
