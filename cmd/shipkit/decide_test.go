package main

import (
	"testing"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		name     string
		plan     *Plan
		expected DecideOutputs
	}{
		{
			name: "verify and tag success - should publish",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "success",
					"test":             "success",
					"integration-test": "success",
					"tag":              "success",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: true,
			},
		},
		{
			name: "dry run mode - no publish",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: true,
				JobResults: map[string]string{
					"build":            "success",
					"test":             "success",
					"integration-test": "success",
					"tag":              "success",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: false,
			},
		},
		{
			name: "build failed - no publish",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "failure",
					"test":             "success",
					"integration-test": "success",
					"tag":              "success",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: false,
			},
		},
		{
			name: "test failed - no publish",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "success",
					"test":             "failure",
					"integration-test": "success",
					"tag":              "success",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: false,
			},
		},
		{
			name: "integration test failed - no publish",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "success",
					"test":             "success",
					"integration-test": "failure",
					"tag":              "success",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: false,
			},
		},
		{
			name: "tag failed - no publish",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "success",
					"test":             "success",
					"integration-test": "success",
					"tag":              "failure",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: false,
			},
		},
		{
			name: "skipped jobs are OK",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "skipped",
					"test":             "skipped",
					"integration-test": "skipped",
					"tag":              "skipped",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: true,
			},
		},
		{
			name: "build skipped, tests success, tag success",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "skipped",
					"test":             "success",
					"integration-test": "success",
					"tag":              "success",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: true,
			},
		},
		{
			name: "verify success, tag skipped",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "success",
					"test":             "success",
					"integration-test": "success",
					"tag":              "skipped",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: true,
			},
		},
		{
			name: "empty results treated as failure",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "",
					"test":             "",
					"integration-test": "",
					"tag":              "",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: false,
			},
		},
		{
			name: "case insensitive success",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            "SUCCESS",
					"test":             "Success",
					"integration-test": "SKIPPED",
					"tag":              "Success",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: true,
			},
		},
		{
			name: "whitespace is trimmed",
			plan: &Plan{
				Mode:   ModeRelease,
				DryRun: false,
				JobResults: map[string]string{
					"build":            " success ",
					"test":             " skipped ",
					"integration-test": " success ",
					"tag":              " success ",
				},
			},
			expected: DecideOutputs{
				ShouldRelease: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Decide(tt.plan)

			if result.ShouldRelease != tt.expected.ShouldRelease {
				t.Errorf("ShouldRelease: got %v, want %v", result.ShouldRelease, tt.expected.ShouldRelease)
			}
		})
	}
}

func TestJobOk(t *testing.T) {
	tests := []struct {
		result string
		want   bool
	}{
		{"success", true},
		{"Success", true},
		{"SUCCESS", true},
		{"skipped", true},
		{"Skipped", true},
		{"SKIPPED", true},
		{" success ", true},
		{" skipped ", true},
		{"failure", false},
		{"Failure", false},
		{"FAILURE", false},
		{"cancelled", false},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.result, func(t *testing.T) {
			got := jobOk(tt.result)
			if got != tt.want {
				t.Errorf("jobOk(%q) = %v, want %v", tt.result, got, tt.want)
			}
		})
	}
}
