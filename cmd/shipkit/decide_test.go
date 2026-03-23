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
			name: "all builds should run and passed - publish all",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               false,
				UseGoreleaser:        true,
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  true,
				ShouldRunDockerBuild: true,
				NpmBuild:             "success",
				GoBuild:              "success",
				MavenBuild:           "success",
				DockerBuild:          "success",
				Tag:                  "success",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    true,
				ShouldPublishMaven:  true,
				ShouldPublishDocker: true,
				ShouldPublishGo:     true,
			},
		},
		{
			name: "dry run mode - no publish",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               true,
				UseGoreleaser:        true,
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  true,
				ShouldRunDockerBuild: true,
				NpmBuild:             "success",
				GoBuild:              "success",
				MavenBuild:           "success",
				DockerBuild:          "success",
				Tag:                  "success",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    false,
				ShouldPublishMaven:  false,
				ShouldPublishDocker: false,
				ShouldPublishGo:     false,
			},
		},
		{
			name: "build failed - no publish",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               false,
				UseGoreleaser:        true,
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  true,
				ShouldRunDockerBuild: true,
				NpmBuild:             "success",
				GoBuild:              "failure",
				MavenBuild:           "success",
				DockerBuild:          "success",
				Tag:                  "success",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    false,
				ShouldPublishMaven:  false,
				ShouldPublishDocker: false,
				ShouldPublishGo:     false,
			},
		},
		{
			name: "tag failed - no publish",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               false,
				UseGoreleaser:        true,
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  true,
				ShouldRunDockerBuild: true,
				NpmBuild:             "success",
				GoBuild:              "success",
				MavenBuild:           "success",
				DockerBuild:          "success",
				Tag:                  "failure",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    false,
				ShouldPublishMaven:  false,
				ShouldPublishDocker: false,
				ShouldPublishGo:     false,
			},
		},
		{
			name: "npm should not run - no npm publish (plan decided)",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               false,
				UseGoreleaser:        true,
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    false, // Plan said don't run npm
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  true,
				ShouldRunDockerBuild: true,
				NpmBuild:             "skipped",
				GoBuild:              "success",
				MavenBuild:           "success",
				DockerBuild:          "success",
				Tag:                  "success",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    false, // Not published because plan said don't run
				ShouldPublishMaven:  true,
				ShouldPublishDocker: true,
				ShouldPublishGo:     true,
			},
		},
		{
			name: "goreleaser mode - no docker publish",
			inputs: DecideInputs{
				Mode:                 ModeGoreleaser,
				DryRun:               false,
				UseGoreleaser:        true,
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  true,
				ShouldRunDockerBuild: true,
				NpmBuild:             "success",
				GoBuild:              "success",
				MavenBuild:           "success",
				DockerBuild:          "success",
				Tag:                  "success",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    true,
				ShouldPublishMaven:  true,
				ShouldPublishDocker: false, // Mode is goreleaser
				ShouldPublishGo:     true,
			},
		},
		{
			name: "goreleaser handles docker - no docker publish",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               false,
				UseGoreleaser:        true,
				GoreleaserDocker:     true, // GoReleaser handles Docker
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  true,
				ShouldRunDockerBuild: false, // Plan didn't run docker-build
				NpmBuild:             "success",
				GoBuild:              "success",
				MavenBuild:           "success",
				DockerBuild:          "skipped",
				Tag:                  "success",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    true,
				ShouldPublishMaven:  true,
				ShouldPublishDocker: false, // GoReleaser handles it
				ShouldPublishGo:     true,
			},
		},
		{
			name: "use_goreleaser disabled - no go publish",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               false,
				UseGoreleaser:        false, // Goreleaser disabled
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  true,
				ShouldRunDockerBuild: true,
				NpmBuild:             "success",
				GoBuild:              "success",
				MavenBuild:           "success",
				DockerBuild:          "success",
				Tag:                  "success",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    true,
				ShouldPublishMaven:  true,
				ShouldPublishDocker: true,
				ShouldPublishGo:     false, // Goreleaser disabled
			},
		},
		{
			name: "skipped jobs are OK",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               false,
				UseGoreleaser:        true,
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  false, // Maven not run
				ShouldRunDockerBuild: true,
				NpmBuild:             "success",
				GoBuild:              "success",
				MavenBuild:           "skipped", // Skipped is OK
				DockerBuild:          "success",
				Tag:                  "skipped", // Skipped is OK
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    true,
				ShouldPublishMaven:  false, // Not published because plan said don't run
				ShouldPublishDocker: true,
				ShouldPublishGo:     true,
			},
		},
		{
			name: "only npm project (plan decided what to run)",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               false,
				UseGoreleaser:        true,
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     false, // Plan said don't run
				ShouldRunMavenBuild:  false, // Plan said don't run
				ShouldRunDockerBuild: false, // Plan said don't run
				NpmBuild:             "success",
				GoBuild:              "skipped",
				MavenBuild:           "skipped",
				DockerBuild:          "skipped",
				Tag:                  "success",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    true,
				ShouldPublishMaven:  false,
				ShouldPublishDocker: false,
				ShouldPublishGo:     true, // Still runs if use_goreleaser=true
			},
		},
		{
			name: "docker should not run (plan decided) - no docker publish",
			inputs: DecideInputs{
				Mode:                 ModeRelease,
				DryRun:               false,
				UseGoreleaser:        true,
				GoreleaserDocker:     false,
				ShouldRunNpmBuild:    true,
				ShouldRunGoBuild:     true,
				ShouldRunMavenBuild:  true,
				ShouldRunDockerBuild: false, // Plan said don't run docker
				NpmBuild:             "success",
				GoBuild:              "success",
				MavenBuild:           "success",
				DockerBuild:          "skipped",
				Tag:                  "success",
				UpdateVersions:       "success",
			},
			expected: DecideOutputs{
				ShouldPublishNpm:    true,
				ShouldPublishMaven:  true,
				ShouldPublishDocker: false, // Not published because plan said don't run
				ShouldPublishGo:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Decide(tt.inputs)

			if result.ShouldPublishNpm != tt.expected.ShouldPublishNpm {
				t.Errorf("ShouldPublishNpm: got %v, want %v", result.ShouldPublishNpm, tt.expected.ShouldPublishNpm)
			}
			if result.ShouldPublishMaven != tt.expected.ShouldPublishMaven {
				t.Errorf("ShouldPublishMaven: got %v, want %v", result.ShouldPublishMaven, tt.expected.ShouldPublishMaven)
			}
			if result.ShouldPublishDocker != tt.expected.ShouldPublishDocker {
				t.Errorf("ShouldPublishDocker: got %v, want %v", result.ShouldPublishDocker, tt.expected.ShouldPublishDocker)
			}
			if result.ShouldPublishGo != tt.expected.ShouldPublishGo {
				t.Errorf("ShouldPublishGo: got %v, want %v", result.ShouldPublishGo, tt.expected.ShouldPublishGo)
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
