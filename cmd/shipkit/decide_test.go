package main

import (
	"testing"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		name     string
		inputs   DecideInputs
		expected DecideOutputs
	}{
		{
			name: "build and tag success - should publish",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: false,
				Build:  "success",
				Tag:    "success",
			},
			expected: DecideOutputs{
				ShouldPublish: true,
			},
		},
		{
			name: "dry run mode - no publish",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: true,
				Build:  "success",
				Tag:    "success",
			},
			expected: DecideOutputs{
				ShouldPublish: false,
			},
		},
		{
			name: "build failed - no publish",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: false,
				Build:  "failure",
				Tag:    "success",
			},
			expected: DecideOutputs{
				ShouldPublish: false,
			},
		},
		{
			name: "tag failed - no publish",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: false,
				Build:  "success",
				Tag:    "failure",
			},
			expected: DecideOutputs{
				ShouldPublish: false,
			},
		},
		{
			name: "skipped jobs are OK",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: false,
				Build:  "skipped",
				Tag:    "skipped",
			},
			expected: DecideOutputs{
				ShouldPublish: true,
			},
		},
		{
			name: "build skipped, tag success",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: false,
				Build:  "skipped",
				Tag:    "success",
			},
			expected: DecideOutputs{
				ShouldPublish: true,
			},
		},
		{
			name: "build success, tag skipped",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: false,
				Build:  "success",
				Tag:    "skipped",
			},
			expected: DecideOutputs{
				ShouldPublish: true,
			},
		},
		{
			name: "empty results treated as failure",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: false,
				Build:  "",
				Tag:    "",
			},
			expected: DecideOutputs{
				ShouldPublish: false,
			},
		},
		{
			name: "case insensitive success",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: false,
				Build:  "SUCCESS",
				Tag:    "Success",
			},
			expected: DecideOutputs{
				ShouldPublish: true,
			},
		},
		{
			name: "whitespace is trimmed",
			inputs: DecideInputs{
				Mode:   ModeRelease,
				DryRun: false,
				Build:  " success ",
				Tag:    " success ",
			},
			expected: DecideOutputs{
				ShouldPublish: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Decide(tt.inputs)

			if result.ShouldPublish != tt.expected.ShouldPublish {
				t.Errorf("ShouldPublish: got %v, want %v", result.ShouldPublish, tt.expected.ShouldPublish)
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
