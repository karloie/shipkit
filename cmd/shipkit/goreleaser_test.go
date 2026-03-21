package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateGoReleaserConfig(t *testing.T) {
	tmpDir := t.TempDir()
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
		HasDocker:    false,
	}

	err := generateGoReleaserConfig(config, outputPath)
	if err != nil {
		t.Fatalf("generateGoReleaserConfig() failed: %v", err)
	}

	// Check file exists
	if !fileExists(outputPath) {
		t.Fatal("output file not created")
	}

	// Check content
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "test-project") {
		t.Error("config should contain project name")
	}
	if !strings.Contains(contentStr, "test-binary") {
		t.Error("config should contain binary name")
	}
	if !strings.Contains(contentStr, "./cmd/test") {
		t.Error("config should contain main path")
	}
}

func TestGenerateGoReleaserConfigWithNodeJS(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.goreleaser.yml")

	config := GoReleaserConfig{
		ProjectName:  "nodejs-project",
		BinaryName:   "nodejs-binary",
		MainPath:     "./cmd/nodejs",
		RepoOwner:    "owner",
		RepoName:     "repo",
		Description:  "Node.js + Go application",
		License:      "Apache-2.0",
		DockerImage:  "owner/nodejs-project",
		HasNodeJS:    true,
		HasChangelog: true,
		HasDocker:    false,
	}

	err := generateGoReleaserConfig(config, outputPath)
	if err != nil {
		t.Fatalf("generateGoReleaserConfig() with Node.js failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	// Check for Node.js hooks
	if !strings.Contains(contentStr, "npm") {
		t.Error("config with HasNodeJS should contain npm commands")
	}
}

func TestGenerateGoReleaserConfigWithDocker(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.goreleaser.yml")

	config := GoReleaserConfig{
		ProjectName:  "docker-project",
		BinaryName:   "docker-binary",
		MainPath:     "./cmd/docker",
		RepoOwner:    "owner",
		RepoName:     "repo",
		Description:  "Dockerized application",
		License:      "MIT",
		DockerImage:  "owner/docker-project",
		HasNodeJS:    false,
		HasChangelog: false,
		HasDocker:    true,
		DockerFile:   "Dockerfile.goreleaser",
	}

	err := generateGoReleaserConfig(config, outputPath)
	if err != nil {
		t.Fatalf("generateGoReleaserConfig() with Docker failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "docker") {
		t.Error("config with HasDocker should contain docker section")
	}
	if !strings.Contains(contentStr, "Dockerfile.goreleaser") {
		t.Error("config should contain specified docker file")
	}
}

func TestDefaultGoReleaserTemplate(t *testing.T) {
	tmpl, err := loadAndParseGoReleaserTemplates()
	if err != nil {
		t.Fatalf("loadAndParseGoReleaserTemplates() failed: %v", err)
	}
	if tmpl == nil {
		t.Error("loadAndParseGoReleaserTemplates() returned nil template")
	}

	// Check that main template exists
	mainTemplate := tmpl.Lookup("goreleaser.yml.tmpl")
	if mainTemplate == nil {
		t.Error("main template goreleaser.yml.tmpl not found")
	}
}

// Snapshot tests to verify goreleaser output
func TestGoReleaserConfigSnapshots(t *testing.T) {
	tests := []struct {
		name       string
		config     GoReleaserConfig
		goldenFile string
	}{
		{
			name: "basic",
			config: GoReleaserConfig{
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
				HasDocker:    false,
			},
			goldenFile: "goreleaser_basic.golden.yml",
		},
		{
			name: "with_nodejs",
			config: GoReleaserConfig{
				ProjectName:  "nodejs-project",
				BinaryName:   "nodejs-binary",
				MainPath:     "./cmd/nodejs",
				RepoOwner:    "owner",
				RepoName:     "repo",
				Description:  "Node.js + Go application",
				License:      "Apache-2.0",
				DockerImage:  "owner/nodejs-project",
				HasNodeJS:    true,
				HasChangelog: true,
				HasDocker:    false,
			},
			goldenFile: "goreleaser_with_nodejs.golden.yml",
		},
		{
			name: "with_docker",
			config: GoReleaserConfig{
				ProjectName:  "docker-project",
				BinaryName:   "docker-binary",
				MainPath:     "./cmd/docker",
				RepoOwner:    "owner",
				RepoName:     "repo",
				Description:  "Dockerized application",
				License:      "MIT",
				DockerImage:  "owner/docker-project",
				HasNodeJS:    false,
				HasChangelog: false,
				HasDocker:    true,
				DockerFile:   "Dockerfile.goreleaser",
			},
			goldenFile: "goreleaser_with_docker.golden.yml",
		},
		{
			name: "full",
			config: GoReleaserConfig{
				ProjectName:  "full-project",
				BinaryName:   "full-binary",
				MainPath:     "./cmd/full",
				RepoOwner:    "full-owner",
				RepoName:     "full-repo",
				Description:  "Full featured application",
				License:      "GPL-3.0",
				DockerImage:  "full-owner/full-project",
				HasNodeJS:    true,
				HasChangelog: true,
				HasDocker:    true,
				DockerFile:   "Containerfile.goreleaser",
			},
			goldenFile: "goreleaser_full.golden.yml",
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
			got, err := os.ReadFile(outputPath)
			if err != nil {
				t.Fatalf("failed to read output: %v", err)
			}

			// Read golden file
			goldenPath := filepath.Join("testdata", tt.goldenFile)
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v", goldenPath, err)
			}

			// Compare
			if string(got) != string(want) {
				t.Errorf("generated config does not match golden file %s\nGot:\n%s\n\nWant:\n%s", tt.goldenFile, string(got), string(want))
			}
		})
	}
}
