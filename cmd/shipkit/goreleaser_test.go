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
	template := defaultGoReleaserTemplate()
	if template == "" {
		t.Error("defaultGoReleaserTemplate() returned empty string")
	}

	// Check essential sections
	if !strings.Contains(template, "builds:") {
		t.Error("template should contain builds section")
	}
	if !strings.Contains(template, "archives:") {
		t.Error("template should contain archives section")
	}
	if !strings.Contains(template, "changelog:") {
		t.Error("template should contain changelog section")
	}
}
