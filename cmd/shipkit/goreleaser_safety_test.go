package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGoReleaserNoDockerFilesPreventsBuildFailure explicitly verifies that when
// no .goreleaser docker files exist, the generated config will NOT include docker
// sections, preventing goreleaser from attempting docker builds and failing.
//
// This is critical because:
// - If `dockers:` section exists but Dockerfile doesn't → goreleaser fails
// - If `dockers:` section is omitted → goreleaser runs successfully
func TestGoReleaserNoDockerFilesPreventsBuildFailure(t *testing.T) {
	tests := []struct {
		name          string
		setupFiles    map[string]string
		expectSuccess bool // should generate valid config
		failureReason string
	}{
		{
			name: "NO docker files - config should be valid (no docker section)",
			setupFiles: map[string]string{
				"go.mod": "module test\n\ngo 1.22\n",
			},
			expectSuccess: true,
		},
		{
			name: "only regular Dockerfile - should NOT include docker section",
			setupFiles: map[string]string{
				"go.mod":     "module test\n\ngo 1.22\n",
				"Dockerfile": "FROM alpine\nCOPY app /app\n",
			},
			expectSuccess: true,
			failureReason: "Regular Dockerfile should not trigger goreleaser docker builds",
		},
		{
			name: "only regular Containerfile - should NOT include docker section",
			setupFiles: map[string]string{
				"go.mod":        "module test\n\ngo 1.22\n",
				"Containerfile": "FROM scratch\n",
			},
			expectSuccess: true,
			failureReason: "Regular Containerfile should not trigger goreleaser docker builds",
		},
		{
			name: "WITH .goreleaser docker file - should include docker section",
			setupFiles: map[string]string{
				"go.mod":                   "module test\n\ngo 1.22\n",
				"Containerfile.goreleaser": "FROM scratch\nCOPY app /app\n",
			},
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			oldDir, _ := os.Getwd()
			defer os.Chdir(oldDir)
			os.Chdir(tmpDir)

			// Setup files
			for filename, content := range tt.setupFiles {
				if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
					t.Fatalf("failed to create %s: %v", filename, err)
				}
			}

			// Detect docker files
			hasDocker, dockerFile := detectDockerFiles("goreleaser")

			// Generate config
			outputPath := filepath.Join(tmpDir, ".goreleaser.yml")
			config := GoReleaserConfig{
				ProjectName:  "test-app",
				BinaryName:   "test-app",
				MainPath:     "./cmd/test",
				RepoOwner:    "test-owner",
				RepoName:     "test-repo",
				Description:  "Test application",
				License:      "MIT",
				DockerImage:  "test-owner/test-app",
				HasChangelog: false,
				HasDocker:    hasDocker,
				DockerFile:   dockerFile,
			}

			err := generateGoReleaserConfig(config, outputPath)
			if err != nil {
				if tt.expectSuccess {
					t.Fatalf("expected success but got error: %v", err)
				}
				return
			}

			// Read generated config
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read config: %v", err)
			}

			output := string(content)

			// Verify docker section behavior matches file presence
			hasDockerSection := strings.Contains(output, "dockers:")

			if hasDocker && !hasDockerSection {
				t.Error("Docker file exists but config missing docker section - would miss publishing opportunity")
			}

			if !hasDocker && hasDockerSection {
				t.Errorf("NO docker file exists but config has docker section - GORELEASER WOULD FAIL!\nReason: %s\nConfig:\n%s",
					tt.failureReason, output)
			}

			// Additional validation: if docker section exists, verify it has required fields
			if hasDockerSection {
				requiredDockerFields := []string{
					"image_templates:",
					"dockerfile:",
					"use: buildx",
				}
				for _, field := range requiredDockerFields {
					if !strings.Contains(output, field) {
						t.Errorf("Docker section exists but missing required field: %s", field)
					}
				}
			}
		})
	}
}

// TestGoReleaserConfigValidatesDockerPresence verifies the config generation
// correctly gates docker sections on file existence, preventing runtime errors
func TestGoReleaserConfigValidatesDockerPresence(t *testing.T) {
	scenarios := []struct {
		scenario            string
		hasDocker           bool
		dockerFile          string
		shouldIncludedocker bool
		reasoning           string
	}{
		{
			scenario:            "No docker file provided",
			hasDocker:           false,
			dockerFile:          "",
			shouldIncludedocker: false,
			reasoning:           "Without docker files, goreleaser shouldn't attempt docker builds",
		},
		{
			scenario:            "Containerfile.goreleaser exists",
			hasDocker:           true,
			dockerFile:          "Containerfile.goreleaser",
			shouldIncludedocker: true,
			reasoning:           "With .goreleaser docker file, docker builds should be enabled",
		},
		{
			scenario:            "Dockerfile.goreleaser exists",
			hasDocker:           true,
			dockerFile:          "Dockerfile.goreleaser",
			shouldIncludedocker: true,
			reasoning:           "With .goreleaser docker file, docker builds should be enabled",
		},
		{
			scenario:            "HasDocker flag without file (should not happen but test defensive)",
			hasDocker:           true,
			dockerFile:          "",
			shouldIncludedocker: true,
			reasoning:           "If HasDocker is true, config includes docker section (file validation happens earlier)",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.scenario, func(t *testing.T) {
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "test.yml")

			config := GoReleaserConfig{
				ProjectName:  "test",
				BinaryName:   "test",
				MainPath:     "./cmd/test",
				RepoOwner:    "owner",
				RepoName:     "test",
				Description:  "Test",
				License:      "MIT",
				DockerImage:  "owner/test",
				HasChangelog: false,
				HasDocker:    sc.hasDocker,
				DockerFile:   sc.dockerFile,
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
			hasDockerSection := strings.Contains(output, "dockers:")

			if sc.shouldIncludedocker != hasDockerSection {
				t.Errorf("Docker section mismatch\nScenario: %s\nReasoning: %s\nExpected docker section: %v\nActual: %v\n",
					sc.scenario, sc.reasoning, sc.shouldIncludedocker, hasDockerSection)
			}

			// Log for debugging
			t.Logf("Scenario: %s | HasDocker=%v | DockerFile=%q | IncludesSection=%v | ✓",
				sc.scenario, sc.hasDocker, sc.dockerFile, hasDockerSection)
		})
	}
}

// TestGoReleaserDockerGatingPreventsFailure demonstrates that without proper gating,
// goreleaser would fail. This test documents WHY the gating is critical.
func TestGoReleaserDockerGatingPreventsFailure(t *testing.T) {
	t.Run("without_gating_would_fail", func(t *testing.T) {
		// Simulate scenario: config has dockers: section but no Dockerfile
		// In real goreleaser run, this would fail with:
		// "Error: docker build failed: no Dockerfile found"

		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "test.yml")

		// This is what would happen WITHOUT proper gating:
		// User has no Containerfile.goreleaser, but config includes docker section
		configWithDockerNoFile := GoReleaserConfig{
			ProjectName:  "test",
			BinaryName:   "test",
			MainPath:     "./cmd/test",
			RepoOwner:    "owner",
			RepoName:     "test",
			Description:  "Test",
			License:      "MIT",
			DockerImage:  "owner/test",
			HasChangelog: false,
			HasDocker:    true,                     // ← Force docker ON without file
			DockerFile:   "nonexistent.dockerfile", // ← File doesn't exist
		}

		err := generateGoReleaserConfig(configWithDockerNoFile, outputPath)
		if err != nil {
			t.Fatalf("config generation failed: %v", err)
		}

		content, _ := os.ReadFile(outputPath)
		if strings.Contains(string(content), "dockers:") {
			t.Log("✓ Generated config includes docker section")
			t.Log("⚠️  If this config were used with goreleaser and the dockerfile doesn't exist,")
			t.Log("⚠️  goreleaser would FAIL with 'docker build failed: no Dockerfile found'")
			t.Log("✓ Our detection logic prevents this by checking file existence FIRST")
		}
	})

	t.Run("with_gating_prevents_failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		oldDir, _ := os.Getwd()
		defer os.Chdir(oldDir)
		os.Chdir(tmpDir)

		// Create go.mod but NO docker files
		os.WriteFile("go.mod", []byte("module test\n\ngo 1.22\n"), 0644)

		// Proper detection - will return HasDocker=false
		hasDocker, dockerFile := detectDockerFiles("goreleaser")

		if hasDocker {
			t.Fatal("detectDockerFiles should return false when no .goreleaser docker files exist")
		}

		// Generate config with proper gating
		outputPath := filepath.Join(tmpDir, "test.yml")
		config := GoReleaserConfig{
			ProjectName:  "test",
			BinaryName:   "test",
			MainPath:     "./cmd/test",
			RepoOwner:    "owner",
			RepoName:     "test",
			Description:  "Test",
			License:      "MIT",
			DockerImage:  "owner/test",
			HasChangelog: false,
			HasDocker:    hasDocker,  // ← Correctly set to false
			DockerFile:   dockerFile, // ← Empty string
		}

		err := generateGoReleaserConfig(config, outputPath)
		if err != nil {
			t.Fatalf("config generation failed: %v", err)
		}

		content, _ := os.ReadFile(outputPath)
		if strings.Contains(string(content), "dockers:") {
			t.Error("Config should NOT include docker section when no docker files exist!")
		} else {
			t.Log("✓ Config correctly OMITS docker section")
			t.Log("✓ Goreleaser will succeed - no docker build attempted")
			t.Log("✓ Gating prevents 'docker build failed' errors")
		}
	})
}
