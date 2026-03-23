package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Docker gating tests verify that Docker configuration in goreleaser.yml
// is conditionally included based on the presence of actual Docker files.
//
// Key behaviors tested:
// 1. Containerfile or Dockerfile triggers Docker section
// 2. Docker section is completely omitted when no Docker files exist
// 3. Preference order: Containerfile > Dockerfile
//
// This ensures goreleaser config only includes Docker publishing when
// Docker files are present.

// TestGoReleaserDockerGating verifies that Docker sections are only included
// when actual Docker files exist (Containerfile or Dockerfile)
func TestGoReleaserDockerGating(t *testing.T) {
	tests := []struct {
		name              string
		setupFiles        map[string]string // files to create
		expectDocker      bool              // should detect docker
		expectDockerFile  string            // expected docker file name
		expectInOutput    []string          // strings that should be in output
		expectNotInOutput []string          // strings that should NOT be in output
	}{
		{
			name:              "no docker files - docker section should be skipped",
			setupFiles:        map[string]string{},
			expectDocker:      false,
			expectDockerFile:  "",
			expectNotInOutput: []string{"dockers:", "image_templates:", "dockerfile:"},
		},
		{
			name: "Containerfile exists - docker section should be included",
			setupFiles: map[string]string{
				"Containerfile": "FROM scratch\nCOPY shipkit /shipkit\n",
			},
			expectDocker:     true,
			expectDockerFile: "Containerfile",
			expectInOutput:   []string{"dockers:", "image_templates:", "dockerfile: Containerfile"},
		},
		{
			name: "Dockerfile exists - docker section should be included",
			setupFiles: map[string]string{
				"Dockerfile": "FROM alpine\nRUN apk add --no-cache ca-certificates\n",
			},
			expectDocker:     true,
			expectDockerFile: "Dockerfile",
			expectInOutput:   []string{"dockers:", "image_templates:", "dockerfile: Dockerfile"},
		},
		{
			name: "both Containerfile and Dockerfile - should prefer Containerfile",
			setupFiles: map[string]string{
				"Containerfile": "FROM scratch\nCOPY shipkit /shipkit\n",
				"Dockerfile":    "FROM alpine\n",
			},
			expectDocker:     true,
			expectDockerFile: "Containerfile",
			expectInOutput:   []string{"dockers:", "dockerfile: Containerfile"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for this test
			tmpDir := t.TempDir()
			oldDir, _ := os.Getwd()
			defer os.Chdir(oldDir)
			os.Chdir(tmpDir)

			// Setup files
			for filename, content := range tt.setupFiles {
				if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
					t.Fatalf("failed to create test file %s: %v", filename, err)
				}
			}

			// Test detection
			hasDocker, dockerFile := detectDockerFiles("goreleaser")
			if hasDocker != tt.expectDocker {
				t.Errorf("detectDockerFiles() hasDocker = %v, want %v", hasDocker, tt.expectDocker)
			}
			if dockerFile != tt.expectDockerFile {
				t.Errorf("detectDockerFiles() dockerFile = %q, want %q", dockerFile, tt.expectDockerFile)
			}

			// Generate goreleaser config
			outputPath := filepath.Join(tmpDir, "test.goreleaser.yml")
			config := GoReleaserConfig{
				ProjectName:  "test-project",
				BinaryName:   "test-binary",
				MainPath:     "./cmd/test",
				RepoOwner:    "test-owner",
				RepoName:     "test-repo",
				Description:  "Test application",
				License:      "MIT",
				DockerImage:  "test-owner/test-project",
				HasNodeJS:    false,
				HasChangelog: false,
				HasDocker:    hasDocker,
				DockerFile:   dockerFile,
			}

			err := generateGoReleaserConfig(config, outputPath)
			if err != nil {
				t.Fatalf("generateGoReleaserConfig() failed: %v", err)
			}

			// Read and verify output
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}
			contentStr := string(content)

			// Check expected strings
			for _, expected := range tt.expectInOutput {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("output should contain %q but does not.\nOutput:\n%s", expected, contentStr)
				}
			}

			// Check strings that should NOT be present
			for _, notExpected := range tt.expectNotInOutput {
				if strings.Contains(contentStr, notExpected) {
					t.Errorf("output should NOT contain %q but does.\nOutput:\n%s", notExpected, contentStr)
				}
			}
		})
	}
}

// TestRunGoReleaserDockerDetection tests that runGoReleaser properly detects
// docker files and sets HasDocker accordingly
func TestRunGoReleaserDockerDetection(t *testing.T) {
	tests := []struct {
		name              string
		setupFiles        map[string]string
		expectDockerMsg   string // expected stderr message
		expectNoDockerMsg string // message when no docker
	}{
		{
			name:              "no docker files",
			setupFiles:        map[string]string{"go.mod": "module test\n\ngo 1.22\n"},
			expectNoDockerMsg: "No Containerfile or Dockerfile found - skipping Docker publishing",
		},
		{
			name: "Containerfile present",
			setupFiles: map[string]string{
				"go.mod":        "module test\n\ngo 1.22\n",
				"Containerfile": "FROM scratch\n",
			},
			expectDockerMsg: "🐳 Detected Containerfile - will publish Docker image",
		},
		{
			name: "Dockerfile present",
			setupFiles: map[string]string{
				"go.mod":     "module test\n\ngo 1.22\n",
				"Dockerfile": "FROM alpine\n",
			},
			expectDockerMsg: "🐳 Detected Dockerfile - will publish Docker image",
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
					t.Fatalf("failed to create test file %s: %v", filename, err)
				}
			}

			// Run goreleaser command (capture would require more complex setup,
			// so we just test the detection logic directly)
			hasDocker, dockerFile := detectDockerFiles("goreleaser")

			if tt.expectDockerMsg != "" {
				if !hasDocker {
					t.Errorf("expected docker to be detected but it was not")
				}
				if dockerFile == "" {
					t.Errorf("expected dockerFile to be set but it was empty")
				}
			}

			if tt.expectNoDockerMsg != "" {
				if hasDocker {
					t.Errorf("expected docker NOT to be detected but it was (file: %s)", dockerFile)
				}
				if dockerFile != "" {
					t.Errorf("expected dockerFile to be empty but it was %q", dockerFile)
				}
			}
		})
	}
}

// TestGoReleaserDockerOutputValidation verifies that the docker template section
// is properly conditionally rendered based on HasDocker flag
func TestGoReleaserDockerOutputValidation(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		hasDocker   bool
		wantMatch   string
		wantNoMatch string
	}{
		{
			name:        "HasDocker=false should not include docker section",
			hasDocker:   false,
			wantNoMatch: "dockers:",
		},
		{
			name:      "HasDocker=true should include docker section",
			hasDocker: true,
			wantMatch: "dockers:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputPath := filepath.Join(tmpDir, tt.name+".yml")
			config := GoReleaserConfig{
				ProjectName:  "test",
				BinaryName:   "test",
				MainPath:     "./cmd/test",
				RepoOwner:    "owner",
				RepoName:     "test",
				Description:  "Test",
				License:      "MIT",
				DockerImage:  "owner/test",
				HasNodeJS:    false,
				HasChangelog: false,
				HasDocker:    tt.hasDocker,
				DockerFile:   "Dockerfile.goreleaser",
			}

			err := generateGoReleaserConfig(config, outputPath)
			if err != nil {
				t.Fatalf("generateGoReleaserConfig() failed: %v", err)
			}

			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output: %v", err)
			}

			contentStr := string(content)

			if tt.wantMatch != "" && !strings.Contains(contentStr, tt.wantMatch) {
				t.Errorf("expected output to contain %q when HasDocker=%v, but it does not", tt.wantMatch, tt.hasDocker)
			}

			if tt.wantNoMatch != "" && strings.Contains(contentStr, tt.wantNoMatch) {
				t.Errorf("expected output NOT to contain %q when HasDocker=%v, but it does", tt.wantNoMatch, tt.hasDocker)
			}
		})
	}
}
