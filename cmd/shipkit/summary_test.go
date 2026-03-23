package main

import (
	"strings"
	"testing"
)

func TestGenerateSummary(t *testing.T) {
	tests := []struct {
		name            string
		inputs          SummaryInputs
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "successful release with all jobs",
			inputs: SummaryInputs{
				Mode:                    "release",
				ToolRef:                 "main",
				Skip:                    false,
				Tag:                     "v1.2.3",
				TagExists:               false,
				Version:                 "1.2.3",
				DockerImage:             "org/app",
				HasGo:                   true,
				HasDocker:               true,
				HasMaven:                true,
				HasNpm:                  true,
				GoreleaserDocker:        false,
				GoreleaserConfigCurrent: true,
				PlanResult:              "success",
				NpmBuildResult:          "success",
				GoBuildResult:           "success",
				MavenBuildResult:        "success",
				DockerBuildResult:       "success",
				TagResult:               "success",
				UpdateVersionsResult:    "success",
				NpmPublishResult:        "success",
				MavenPublishResult:      "success",
				DockerPublishResult:     "success",
				GoPublishResult:         "success",
			},
			wantContains: []string{
				"# 🚀 Release Summary",
				"## 📋 Plan",
				"| Mode | `release` |",
				"| Tool Ref | `main` |",
				"| Tag | `v1.2.3` |",
				"| Version | `1.2.3` |",
				"| Docker Image | `org/app` |",
				"## 🔍 Detected Projects",
				"| Go | ✅ |",
				"| Docker | ✅ | ❌ |",
				"| Maven | ✅ |",
				"| npm | ✅ |",
				"✅ Using **custom** .goreleaser.yml config",
				"## ⚙️ Execution Results",
				"| 🚢 Plan | ✅ Success |",
				"| 🏗️ npm Build | ✅ Success |",
				"| 🏗️ Go Build | ✅ Success |",
				"| 🏗️ Maven Build | ✅ Success |",
				"| 🏗️ Docker Build | ✅ Success |",
				"| 🏷️ Tag | ✅ Success |",
				"| 📝 Update Versions | ✅ Success |",
				"| 🚀 npm Publish | ✅ Success |",
				"| 🚀 Maven Publish | ✅ Success |",
				"| 🚀 Docker Publish | ✅ Success |",
				"| 🚀 Go Publish | ✅ Success |",
				"## ✅ Overall Status: **SUCCESS**",
				"Release `v1.2.3` completed successfully!",
			},
			wantNotContains: []string{
				"❌ Failed",
				"SKIPPED",
				"PARTIAL",
			},
		},
		{
			name: "skipped release",
			inputs: SummaryInputs{
				Mode:       "release",
				ToolRef:    "main",
				Skip:       true,
				Tag:        "",
				PlanResult: "success",
			},
			wantContains: []string{
				"## ℹ️ Overall Status: **SKIPPED**",
				"No release markers found. Release was skipped.",
			},
			wantNotContains: []string{
				"SUCCESS",
				"FAILED",
			},
		},
		{
			name: "plan failed",
			inputs: SummaryInputs{
				Mode:       "release",
				ToolRef:    "main",
				Skip:       false,
				Tag:        "v1.0.0",
				PlanResult: "failure",
			},
			wantContains: []string{
				"| 🚢 Plan | ❌ Failed |",
				"## ❌ Overall Status: **FAILED**",
				"Release failed during planning phase.",
			},
			wantNotContains: []string{
				"SUCCESS",
				"SKIPPED",
			},
		},
		{
			name: "tag failed",
			inputs: SummaryInputs{
				Mode:          "release",
				ToolRef:       "main",
				Skip:          false,
				Tag:           "v1.0.0",
				PlanResult:    "success",
				GoBuildResult: "success",
				TagResult:     "failure",
			},
			wantContains: []string{
				"| 🏷️ Tag | ❌ Failed |",
				"## ❌ Overall Status: **FAILED**",
				"Release failed during tag creation.",
			},
		},
		{
			name: "partial success",
			inputs: SummaryInputs{
				Mode:             "release",
				ToolRef:          "main",
				Skip:             false,
				Tag:              "v1.0.0",
				PlanResult:       "success",
				GoBuildResult:    "success",
				TagResult:        "success",
				GoPublishResult:  "skipped",
				NpmPublishResult: "skipped",
			},
			wantContains: []string{
				"## ⚠️ Overall Status: **PARTIAL**",
				"Some jobs succeeded, but publish phase may have been skipped or failed.",
			},
		},
		{
			name: "auto-generated goreleaser config",
			inputs: SummaryInputs{
				Mode:                    "release",
				ToolRef:                 "main",
				GoreleaserConfigCurrent: false,
				PlanResult:              "success",
			},
			wantContains: []string{
				"🔧 Will **auto-generate** GoReleaser config",
			},
			wantNotContains: []string{
				"custom",
			},
		},
		{
			name: "only show jobs that ran",
			inputs: SummaryInputs{
				Mode:              "release",
				ToolRef:           "main",
				Skip:              false,
				PlanResult:        "success",
				GoBuildResult:     "success",
				NpmBuildResult:    "skipped",
				MavenBuildResult:  "skipped",
				DockerBuildResult: "skipped",
				TagResult:         "success",
				GoPublishResult:   "success",
			},
			wantContains: []string{
				"| 🚢 Plan | ✅ Success |",
				"| 🏗️ Go Build | ✅ Success |",
				"| 🏷️ Tag | ✅ Success |",
				"| 🚀 Go Publish | ✅ Success |",
			},
			wantNotContains: []string{
				"npm Build",
				"Maven Build",
				"Docker Build",
			},
		},
		{
			name: "goreleaser handles docker",
			inputs: SummaryInputs{
				Mode:             "release",
				ToolRef:          "main",
				HasDocker:        true,
				GoreleaserDocker: true,
				PlanResult:       "success",
			},
			wantContains: []string{
				"| Docker | ✅ | ✅ |",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSummary(tt.inputs)

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Summary missing expected content: %q\nGot:\n%s", want, result)
				}
			}

			for _, notWant := range tt.wantNotContains {
				if strings.Contains(result, notWant) {
					t.Errorf("Summary contains unexpected content: %q\nGot:\n%s", notWant, result)
				}
			}
		})
	}
}

func TestCheckmark(t *testing.T) {
	tests := []struct {
		value bool
		want  string
	}{
		{true, "✅"},
		{false, "❌"},
	}

	for _, tt := range tests {
		got := checkmark(tt.value)
		if got != tt.want {
			t.Errorf("checkmark(%v) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestStatusBadge(t *testing.T) {
	tests := []struct {
		result string
		want   string
	}{
		{"success", "✅ Success"},
		{"Success", "✅ Success"},
		{"SUCCESS", "✅ Success"},
		{"failure", "❌ Failed"},
		{"Failure", "❌ Failed"},
		{"skipped", "⏭️ Skipped"},
		{"cancelled", "🚫 Cancelled"},
		{" success ", "✅ Success"},
		{"unknown", "⚠️ unknown"},
	}

	for _, tt := range tests {
		got := statusBadge(tt.result)
		if got != tt.want {
			t.Errorf("statusBadge(%q) = %q, want %q", tt.result, got, tt.want)
		}
	}
}

func TestJobRan(t *testing.T) {
	tests := []struct {
		result string
		want   bool
	}{
		{"success", true},
		{"Success", true},
		{"failure", true},
		{"Failure", true},
		{" success ", true},
		{" failure ", true},
		{"skipped", false},
		{"cancelled", false},
		{"", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		got := jobRan(tt.result)
		if got != tt.want {
			t.Errorf("jobRan(%q) = %v, want %v", tt.result, got, tt.want)
		}
	}
}

func TestDetermineOverallStatus(t *testing.T) {
	tests := []struct {
		name         string
		inputs       SummaryInputs
		wantContains string
	}{
		{
			name: "skipped",
			inputs: SummaryInputs{
				Skip: true,
			},
			wantContains: "SKIPPED",
		},
		{
			name: "success - go published",
			inputs: SummaryInputs{
				Skip:            false,
				Tag:             "v1.0.0",
				GoPublishResult: "success",
			},
			wantContains: "SUCCESS",
		},
		{
			name: "success - docker published",
			inputs: SummaryInputs{
				Skip:                false,
				Tag:                 "v1.0.0",
				DockerPublishResult: "success",
			},
			wantContains: "SUCCESS",
		},
		{
			name: "plan failed",
			inputs: SummaryInputs{
				Skip:       false,
				PlanResult: "failure",
			},
			wantContains: "planning phase",
		},
		{
			name: "tag failed",
			inputs: SummaryInputs{
				Skip:      false,
				TagResult: "failure",
			},
			wantContains: "tag creation",
		},
		{
			name: "partial",
			inputs: SummaryInputs{
				Skip:       false,
				PlanResult: "success",
				TagResult:  "success",
			},
			wantContains: "PARTIAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineOverallStatus(tt.inputs)
			if !strings.Contains(result, tt.wantContains) {
				t.Errorf("determineOverallStatus() missing %q\nGot: %s", tt.wantContains, result)
			}
		})
	}
}
