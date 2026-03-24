package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoReleaserConfigMatrix tests a comprehensive matrix of configuration
// combinations to ensure templates handle all permutations correctly.
// This is a "rolling" matrix that tests progressive feature additions.
func TestGoReleaserConfigMatrix(t *testing.T) {
	// Test matrix: progressively add features
	tests := []struct {
		name           string
		config         GoReleaserConfig
		mustContain    []string // strings that MUST be present
		mustNotContain []string // strings that MUST NOT be present
		minLineCount   int      // minimum expected lines in output
		maxLineCount   int      // maximum expected lines in output
	}{
		{
			name: "minimal - all features off",
			config: GoReleaserConfig{
				ProjectName:  "minimal",
				BinaryName:   "minimal",
				MainPath:     "./cmd/minimal",
				RepoOwner:    "owner",
				RepoName:     "minimal",
				Description:  "Minimal config",
				License:      "MIT",
				DockerImage:  "owner/minimal",
				HasChangelog: false,
				HasDocker:    false,
			},
			mustContain: []string{
				"version: 2",
				"builds:",
				"archives:",
				"nfpms:",
				"homebrew_casks:",
				"changelog:",
				"disable: true",
			},
			mustNotContain: []string{
				"npm ci",
				"npm run build",
				"dockers:",
				"use: github",
			},
			minLineCount: 60,
			maxLineCount: 90,
		},
		{
			name: "changelog only",
			config: GoReleaserConfig{
				ProjectName:  "changelog-app",
				BinaryName:   "changelog-app",
				MainPath:     "./cmd/changelog-app",
				RepoOwner:    "owner",
				RepoName:     "changelog-app",
				Description:  "Changelog config",
				License:      "GPL-3.0",
				DockerImage:  "owner/changelog-app",
				HasChangelog: true,
				HasDocker:    false,
			},
			mustContain: []string{
				"changelog:",
				"use: github",
			},
			mustNotContain: []string{
				"npm ci",
				"dockers:",
				"disable: true",
			},
			minLineCount: 60,
			maxLineCount: 90,
		},
		{
			name: "docker only - Containerfile",
			config: GoReleaserConfig{
				ProjectName:  "docker-app",
				BinaryName:   "docker-app",
				MainPath:     "./cmd/docker-app",
				RepoOwner:    "owner",
				RepoName:     "docker-app",
				Description:  "Docker config",
				License:      "MIT",
				DockerImage:  "owner/docker-app",
				HasChangelog: false,
				HasDocker:    true,
				DockerFile:   "Containerfile.goreleaser",
			},
			mustContain: []string{
				"dockers:",
				"dockerfile: Containerfile.goreleaser",
				"image_templates:",
				"org.opencontainers.image",
				"disable: true",
			},
			mustNotContain: []string{
				"npm ci",
				"use: github",
			},
			minLineCount: 75,
			maxLineCount: 105,
		},
		{
			name: "docker only - Dockerfile",
			config: GoReleaserConfig{
				ProjectName:  "docker-app2",
				BinaryName:   "docker-app2",
				MainPath:     "./cmd/docker-app2",
				RepoOwner:    "owner",
				RepoName:     "docker-app2",
				Description:  "Docker config with Dockerfile",
				License:      "BSD-3-Clause",
				DockerImage:  "owner/docker-app2",
				HasChangelog: false,
				HasDocker:    true,
				DockerFile:   "Dockerfile.goreleaser",
			},
			mustContain: []string{
				"dockers:",
				"dockerfile: Dockerfile.goreleaser",
			},
			mustNotContain: []string{
				"npm ci",
			},
			minLineCount: 75,
			maxLineCount: 105,
		},
		{
			name: "changelog + docker",
			config: GoReleaserConfig{
				ProjectName:  "changelog-docker",
				BinaryName:   "changelog-docker",
				MainPath:     "./cmd/changelog-docker",
				RepoOwner:    "owner",
				RepoName:     "changelog-docker",
				Description:  "Changelog with Docker",
				License:      "MIT",
				DockerImage:  "owner/changelog-docker",
				HasChangelog: true,
				HasDocker:    true,
				DockerFile:   "Dockerfile.goreleaser",
			},
			mustContain: []string{
				"dockers:",
				"use: github",
			},
			mustNotContain: []string{
				"npm ci",
				"disable: true",
			},
			minLineCount: 75,
			maxLineCount: 115,
		},
		{
			name: "all features enabled - maximal",
			config: GoReleaserConfig{
				ProjectName:  "maximal",
				BinaryName:   "maximal",
				MainPath:     "./cmd/maximal",
				RepoOwner:    "maximal-owner",
				RepoName:     "maximal-repo",
				Description:  "All features enabled",
				License:      "Apache-2.0",
				DockerImage:  "maximal-owner/maximal",
				HasChangelog: true,
				HasDocker:    true,
				DockerFile:   "Containerfile.goreleaser",
			},
			mustContain: []string{
				"dockers:",
				"use: github",
			},
			mustNotContain: []string{
				"disable: true",
			},
			minLineCount: 85,
			maxLineCount: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "output.yml")

			// Generate config
			err := generateGoReleaserConfig(tt.config, outputPath)
			if err != nil {
				t.Fatalf("generateGoReleaserConfig() failed: %v", err)
			}

			// Read generated output
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output: %v", err)
			}

			contentStr := string(content)
			lines := strings.Split(contentStr, "\n")

			// Check line count
			lineCount := len(lines)
			if lineCount < tt.minLineCount {
				t.Errorf("output has %d lines, expected at least %d", lineCount, tt.minLineCount)
			}
			if lineCount > tt.maxLineCount {
				t.Errorf("output has %d lines, expected at most %d", lineCount, tt.maxLineCount)
			}

			// Check required strings
			for _, required := range tt.mustContain {
				if !strings.Contains(contentStr, required) {
					t.Errorf("output missing required string %q\nConfig: %+v\nOutput:\n%s",
						required, tt.config, contentStr)
				}
			}

			// Check forbidden strings
			for _, forbidden := range tt.mustNotContain {
				if strings.Contains(contentStr, forbidden) {
					t.Errorf("output contains forbidden string %q\nConfig: %+v\nOutput:\n%s",
						forbidden, tt.config, contentStr)
				}
			}

			// Verify basic structure
			if !strings.HasPrefix(contentStr, "version: 2") {
				t.Error("output should start with 'version: 2'")
			}

			// Verify all configs have required sections
			requiredSections := []string{"builds:", "archives:", "nfpms:", "homebrew_casks:", "release:", "changelog:"}
			for _, section := range requiredSections {
				if !strings.Contains(contentStr, section) {
					t.Errorf("output missing required section %q", section)
				}
			}
		})
	}
}

// TestGoReleaserConfigEdgeCases tests edge cases and boundary conditions
func TestGoReleaserConfigEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		config GoReleaserConfig
		verify func(t *testing.T, output string)
	}{
		{
			name: "empty strings should not break template",
			config: GoReleaserConfig{
				ProjectName:  "test",
				BinaryName:   "test",
				MainPath:     "./cmd/test",
				RepoOwner:    "owner",
				RepoName:     "test",
				Description:  "", // empty description
				License:      "MIT",
				DockerImage:  "owner/test",
				HasChangelog: false,
				HasDocker:    false,
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "description:") {
					t.Error("should still have description field even if empty")
				}
			},
		},
		{
			name: "special characters in project name",
			config: GoReleaserConfig{
				ProjectName:  "test-project_v2",
				BinaryName:   "test-binary",
				MainPath:     "./cmd/test-project",
				RepoOwner:    "test-owner",
				RepoName:     "test-repo",
				Description:  "Test with special chars: @#$%",
				License:      "MIT",
				DockerImage:  "owner/test",
				HasChangelog: false,
				HasDocker:    false,
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "test-project_v2") {
					t.Error("should preserve special characters in project name")
				}
			},
		},
		{
			name: "different main paths",
			config: GoReleaserConfig{
				ProjectName:  "test",
				BinaryName:   "test",
				MainPath:     ".", // root directory
				RepoOwner:    "owner",
				RepoName:     "test",
				Description:  "Root main",
				License:      "MIT",
				DockerImage:  "owner/test",
				HasChangelog: false,
				HasDocker:    false,
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "main: .") {
					t.Error("should handle root directory as main path")
				}
			},
		},
		{
			name: "various licenses",
			config: GoReleaserConfig{
				ProjectName:  "test",
				BinaryName:   "test",
				MainPath:     "./cmd/test",
				RepoOwner:    "owner",
				RepoName:     "test",
				Description:  "Test",
				License:      "AGPL-3.0-only",
				DockerImage:  "owner/test",
				HasChangelog: false,
				HasDocker:    false,
			},
			verify: func(t *testing.T, output string) {
				if !strings.Contains(output, "AGPL-3.0-only") {
					t.Error("should preserve exact license string")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "output.yml")

			err := generateGoReleaserConfig(tt.config, outputPath)
			if err != nil {
				t.Fatalf("generateGoReleaserConfig() failed: %v", err)
			}

			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output: %v", err)
			}

			tt.verify(t, string(content))
		})
	}
}

// TestGoReleaserConfigCombinations tests specific feature combinations
// in a rolling pattern to ensure orthogonality
func TestGoReleaserConfigCombinations(t *testing.T) {
	// Build all 2^3 = 8 combinations of the three boolean flags
	type flags struct {
		nodejs    bool
		changelog bool
		docker    bool
	}

	combinations := []flags{
		{false, false, false}, // 000
		{true, false, false},  // 100
		{false, true, false},  // 010
		{false, false, true},  // 001
		{true, true, false},   // 110
		{true, false, true},   // 101
		{false, true, true},   // 011
		{true, true, true},    // 111
	}

	for i, combo := range combinations {
		name := fmt.Sprintf("combo_%d_nodejs=%v_changelog=%v_docker=%v",
			i, combo.nodejs, combo.changelog, combo.docker)

		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "output.yml")

			config := GoReleaserConfig{
				ProjectName:  fmt.Sprintf("test-%d", i),
				BinaryName:   fmt.Sprintf("test-%d", i),
				MainPath:     "./cmd/test",
				RepoOwner:    "owner",
				RepoName:     "test",
				Description:  "Test",
				License:      "MIT",
				DockerImage:  "owner/test",
				HasChangelog: combo.changelog,
				HasDocker:    combo.docker,
				DockerFile:   "Containerfile.goreleaser",
			}

			err := generateGoReleaserConfig(config, outputPath)
			if err != nil {
				t.Fatalf("generateGoReleaserConfig() failed: %v", err)
			}

			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output: %v", err)
			}

			output := string(content)

			// Verify changelog flag behavior (opt-in: create .goreleaser-changelog to enable)
			if combo.changelog {
				if !strings.Contains(output, "use: github") {
					t.Error("HasChangelog=true should enable github auto-changelog (opted in)")
				}
			} else {
				if !strings.Contains(output, "disable: true") {
					t.Error("HasChangelog=false should disable auto-changelog (default)")
				}
			}

			// Verify docker flag behavior
			if combo.docker {
				if !strings.Contains(output, "dockers:") {
					t.Error("HasDocker=true should include dockers section")
				}
			} else {
				if strings.Contains(output, "dockers:") {
					t.Error("HasDocker=false should not include dockers section")
				}
			}

			// All configs should have core sections
			if !strings.Contains(output, "builds:") {
				t.Error("all configs should have builds section")
			}
		})
	}
}
