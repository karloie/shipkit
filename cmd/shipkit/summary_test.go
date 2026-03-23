package main

import (
	"strings"
	"testing"
)

func TestGenerateSummary(t *testing.T) {
	inputs := SummaryInputs{
		Mode:                    ModeRelease,
		ToolRef:                 "main",
		Tag:                     "v1.2.3",
		VersionClean:            "1.2.3",
		HasGo:                   true,
		BuildOrchestrator:       "make",
		GoreleaserConfigCurrent: true,
		ResultPlan:              "success",
		ResultBuild:             "success",
		ResultTag:               "success",
		ResultPublish:           "success",
	}

	result := GenerateSummary(inputs)

	tests := []string{"Release Summary", "v1.2.3", "Make", "success"}
	for _, expected := range tests {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected summary to contain %q", expected)
		}
	}
}

func TestStatusBadge(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"success", "✅ success"},
		{"failure", "❌ failure"},
		{"skipped", "⏭️ skipped"},
	}

	for _, tt := range tests {
		got := statusBadge(tt.status)
		if got != tt.want {
			t.Errorf("statusBadge(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestCheckmark(t *testing.T) {
	if checkmark(true) != "✅" {
		t.Error("checkmark(true) should return ✅")
	}
	if checkmark(false) != "❌" {
		t.Error("checkmark(false) should return ❌")
	}
}
