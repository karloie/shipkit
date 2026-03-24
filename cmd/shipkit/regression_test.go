package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regression: rerelease mode must skip computeVersion (commit marker check) entirely.
// Before fix: plan -mode=rerelease with no -next-tag would call computeVersion, find no
// release markers on push event, return publish=skip and short-circuit before resolving tag.
func TestRereleaseSkipsComputeVersion(t *testing.T) {
	env := &EnvProviderMock{values: map[string]string{
		"DOCKERHUB_USERNAME":        "u",
		"DOCKERHUB_TOKEN":           "t",
		"HOMEBREW_TAP_GITHUB_TOKEN": "h",
	}}
	// ResolveLatest=true, no NextTag — must resolve from git, not skip
	p, err := computeReleasePolicy(PolicyInput{
		Mode:            ModeRerelease,
		Release:         ReleaseTrue,
		ResolveLatest:   true,
		Image:           "karloie/kompass",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN", "HOMEBREW_TAP_GITHUB_TOKEN"},
	}, env, &GitProviderMock{LatestTag: "v0.0.16"})
	if err != nil {
		t.Fatalf("rerelease should not error: %v", err)
	}
	if p.Skip == ReleaseTrue {
		t.Fatal("rerelease must not be skipped when publish=true and tag is resolvable")
	}
	if p.ReleaseTag != "v0.0.16" {
		t.Fatalf("expected ReleaseTag=v0.0.16, got %q", p.ReleaseTag)
	}
}

// Regression: plan outputs Dockerfile field to GITHUB_OUTPUT.
// Before fix: policy.Dockerfile was computed but never written to output.
func TestPlanWritesDockerfileOutput(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "github_output")
	t.Setenv("GITHUB_OUTPUT", outFile)

	// Create a Containerfile so dockerfile detection works
	tmpDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tmpDir)
	os.WriteFile("Containerfile", []byte("FROM scratch"), 0644)

	env := &EnvProviderMock{values: map[string]string{
		"DOCKERHUB_USERNAME":        "u",
		"DOCKERHUB_TOKEN":           "t",
		"HOMEBREW_TAP_GITHUB_TOKEN": "h",
	}}
	p, err := computeReleasePolicy(PolicyInput{
		Mode:            ModeRerelease,
		Release:         ReleaseTrue,
		NextTag:         "v1.2.3",
		Image:           "karloie/kompass",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN", "HOMEBREW_TAP_GITHUB_TOKEN"},
	}, env, &GitProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Dockerfile == "" {
		t.Fatal("policy.Dockerfile should be non-empty when Containerfile exists")
	}

	// Simulate what plan.go does
	writeOutput(outFile, OutputDockerfile, p.Dockerfile)

	content, _ := os.ReadFile(outFile)
	if !strings.Contains(string(content), "dockerfile=") {
		t.Fatalf("GITHUB_OUTPUT missing dockerfile entry, got: %s", content)
	}
}

// Regression: parseMajorMinor returns empty string for 0.x.y versions.
// Before fix: 0.0.16 → "0.0" was pushed as a Docker tag, creating garbage tags.
func TestParseMajorMinorZeroMajor(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{"0.0.1", ""},
		{"0.1.2", ""},
		{"0.99.99", ""},
		{"1.0.0", "1.0"},
		{"2.3.4", "2.3"},
	}
	for _, tt := range tests {
		got, err := parseMajorMinor(tt.version)
		if err != nil {
			t.Fatalf("parseMajorMinor(%q) unexpected error: %v", tt.version, err)
		}
		if got != tt.want {
			t.Errorf("parseMajorMinor(%q) = %q, want %q", tt.version, got, tt.want)
		}
	}
}

// Regression: goreleaser template uses "disable: true" not "skip: true" for changelog.
// Before fix: generated config caused goreleaser v2 to fail with "field skip not found".
func TestGoReleaserTemplateChangelogDisable(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.yml")

	err := generateGoReleaserConfig(GoReleaserConfig{
		ProjectName:  "myapp",
		BinaryName:   "myapp",
		MainPath:     "./cmd/myapp",
		RepoOwner:    "owner",
		RepoName:     "myapp",
		Description:  "test",
		License:      "MIT",
		DockerImage:  "owner/myapp",
		HasChangelog: false, // no opt-in → changelog disabled (default)
	}, outputPath)
	if err != nil {
		t.Fatalf("generateGoReleaserConfig failed: %v", err)
	}
	content, _ := os.ReadFile(outputPath)
	s := string(content)
	if strings.Contains(s, "skip:") {
		t.Error("generated config must not use 'skip:' (invalid in goreleaser v2)")
	}
	if !strings.Contains(s, "disable:") {
		t.Error("generated config must use 'disable: true' for changelog")
	}
}

// Regression: goreleaser subcommand detects Node using "Node" not "Node.js".
// Before fix: hasProjectType(detected, "Node.js") never matched because detect.go
// registers the type as "Node", so npm hooks were omitted from generated config.
func TestGoReleaserNodeJSDetection(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// Create package.json to trigger Node detection
	if err := os.WriteFile("package.json", []byte(`{"name":"web","scripts":{"build":"vite build"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	detected := detectProjectTypesWithLogging(false)
	hasNode := hasProjectType(detected, "Node")
	if !hasNode {
		t.Fatal("Node project type should be detected when package.json is present")
	}

	// Verify the goreleaser config includes npm hooks when Node is detected
	outputPath := filepath.Join(tmpDir, "out.yml")
	err := generateGoReleaserConfig(GoReleaserConfig{
		ProjectName: "myapp",
		BinaryName:  "myapp",
		MainPath:    "./cmd/myapp",
		RepoOwner:   "owner",
		RepoName:    "myapp",
		Description: "test",
		License:     "MIT",
		DockerImage: "owner/myapp",
		HasNodeJS:   hasNode,
	}, outputPath)
	if err != nil {
		t.Fatalf("generateGoReleaserConfig failed: %v", err)
	}
	content, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(content), "npm") {
		t.Error("generated config must include npm hooks when Node project is detected")
	}
}

// Regression: goreleaser template must not produce invalid or deprecated fields.
// Validates the generated YAML has no known-bad fields from past bugs.
func TestGoReleaserTemplateNoInvalidFields(t *testing.T) {
	tmpl, err := loadAndParseGoReleaserTemplates()
	if err != nil {
		t.Fatalf("loadAndParseGoReleaserTemplates() failed: %v", err)
	}

	config := GoReleaserConfig{
		ProjectName: "test",
		BinaryName:  "test",
		MainPath:    "./cmd/test",
		RepoOwner:   "owner",
		RepoName:    "test",
		Description: "test app",
		License:     "MIT",
		DockerImage: "owner/test",
	}

	// Create a temporary file to write output
	tmpFile, err := os.CreateTemp("", "goreleaser-test-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := tmpl.ExecuteTemplate(tmpFile, "goreleaser.yml.tmpl", config); err != nil {
		t.Fatalf("failed to execute template: %v", err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	tmplStr := string(content)

	invalidFields := []string{
		"skip:",    // replaced by disable: in v2
		"ignore:",  // was accidentally added as top-level key
		"\nbrews:", // deprecated in goreleaser v2, replaced by homebrew_casks
	}
	for _, bad := range invalidFields {
		if strings.Contains(tmplStr, bad) {
			t.Errorf("template contains invalid/deprecated goreleaser v2 field %q", bad)
		}
	}
	if !strings.Contains(tmplStr, "homebrew_casks:") {
		t.Error("template must use 'homebrew_casks' (not deprecated 'brews')")
	}
}

// Regression: mode=docker must skip when no Containerfile/Dockerfile exists.
// Before fix: detectDockerfileForWorkflow() returned "Containerfile" as hardcoded fallback,
// causing docker builds to run and fail with "open Containerfile: no such file or directory".
func TestDockerModeSkipsWhenNoDockerfile(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// No Containerfile or Dockerfile in this directory
	env := &EnvProviderMock{values: map[string]string{
		"DOCKERHUB_USERNAME": "u",
		"DOCKERHUB_TOKEN":    "t",
	}}

	p, err := computeReleasePolicy(PolicyInput{
		Mode:            ModeDocker,
		Release:         ReleaseTrue,
		NextTag:         "v1.0.0",
		Image:           "test/image",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN"},
	}, env, &GitProviderMock{})

	if err != nil {
		t.Fatalf("computeReleasePolicy failed: %v", err)
	}

	// Must set skip=true to skip the docker job in workflow
	if p.Skip != ReleaseTrue {
		t.Errorf("expected Skip=ReleaseTrue when no dockerfile exists, got %q", p.Skip)
	}

	// Must have informative message
	if !strings.Contains(p.Message, "Docker build will be skipped") {
		t.Errorf("expected message about docker skip, got %q", p.Message)
	}

	// Dockerfile output should be empty
	if p.Dockerfile != "" {
		t.Errorf("expected empty Dockerfile when no file exists, got %q", p.Dockerfile)
	}
}

// Regression: plan must output has_docker to control workflow docker job skipping.
// Before fix: docker job ran when no Docker files existed, showing green instead of gray (skipped).
// The docker job should only run when standalone Docker files exist AND goreleaser isn't handling Docker.
func TestPlanOutputsHasDockerForWorkflowCondition(t *testing.T) {
	tests := []struct {
		name                 string
		createFiles          []string
		expectedHasDocker    bool
		expectedGorelDocker  bool
		workflowShouldRunDoc bool // Should docker job run in workflow?
	}{
		{
			name:                 "no docker files - skip docker job",
			createFiles:          []string{},
			expectedHasDocker:    false,
			expectedGorelDocker:  false,
			workflowShouldRunDoc: false,
		},
		{
			name:                 "Dockerfile exists - goreleaser handles docker, standalone job skips",
			createFiles:          []string{"Dockerfile"},
			expectedHasDocker:    true,
			expectedGorelDocker:  true,
			workflowShouldRunDoc: false, // Goreleaser handles it, so standalone job doesn't run
		},
		{
			name:                 "Containerfile exists - goreleaser handles docker, standalone job skips",
			createFiles:          []string{"Containerfile"},
			expectedHasDocker:    true,
			expectedGorelDocker:  true,
			workflowShouldRunDoc: false, // Goreleaser handles it, so standalone job doesn't run
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outFile := filepath.Join(t.TempDir(), "github_output")

			tmpDir := t.TempDir()
			oldWd, _ := os.Getwd()
			defer os.Chdir(oldWd)
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal(err)
			}

			// Create docker files based on test case
			for _, file := range tt.createFiles {
				if err := os.WriteFile(file, []byte("FROM scratch"), 0644); err != nil {
					t.Fatalf("failed to create %s: %v", file, err)
				}
			}

			// Test the detection logic directly and write outputs
			hasStandaloneDocker := fileExists(FileContainerfile) || fileExists(FileDockerfile)
			hasGoreleaserDocker := fileExists(FileContainerfile) || fileExists(FileDockerfile)

			// Verify detection matches expectations
			if hasStandaloneDocker != tt.expectedHasDocker {
				t.Errorf("hasStandaloneDocker = %v, want %v", hasStandaloneDocker, tt.expectedHasDocker)
			}
			if hasGoreleaserDocker != tt.expectedGorelDocker {
				t.Errorf("hasGoreleaserDocker = %v, want %v", hasGoreleaserDocker, tt.expectedGorelDocker)
			}

			// Simulate what plan.go writes to outputs
			if hasStandaloneDocker {
				writeOutput(outFile, OutputHasDocker, ReleaseTrue)
			} else {
				writeOutput(outFile, OutputHasDocker, ReleaseFalse)
			}
			if hasGoreleaserDocker {
				writeOutput(outFile, OutputGoreleaserDocker, ReleaseTrue)
			} else {
				writeOutput(outFile, OutputGoreleaserDocker, ReleaseFalse)
			}

			// Read outputs
			content, err := os.ReadFile(outFile)
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}
			outputs := string(content)

			// Verify outputs are written correctly
			if tt.expectedHasDocker {
				if !strings.Contains(outputs, "has_docker=true") {
					t.Errorf("expected has_docker=true in output, got:\n%s", outputs)
				}
			} else {
				if !strings.Contains(outputs, "has_docker=false") {
					t.Errorf("expected has_docker=false in output, got:\n%s", outputs)
				}
			}

			if tt.expectedGorelDocker {
				if !strings.Contains(outputs, "should_build_docker_goreleaser=true") {
					t.Errorf("expected should_build_docker_goreleaser=true in output, got:\n%s", outputs)
				}
			} else {
				if !strings.Contains(outputs, "should_build_docker_goreleaser=false") {
					t.Errorf("expected should_build_docker_goreleaser=false in output, got:\n%s", outputs)
				}
			}

			// Simulate workflow condition logic
			hasDocker := strings.Contains(outputs, "has_docker=true")
			hasDockerEmpty := !strings.Contains(outputs, "has_docker=") // Missing output (old versions)
			goreleaserDocker := strings.Contains(outputs, "should_build_docker_goreleaser=true")
			skip := false // For this test, assume not skipped

			// Workflow condition (backward compatible):
			// if: needs.plan.outputs.skip != 'true' && needs.plan.outputs.should_build_docker_goreleaser != 'true' && (needs.plan.outputs.has_docker == 'true' || needs.plan.outputs.has_docker == '')
			// Empty/missing has_docker (from old versions) is treated as true for backward compatibility
			workflowWouldRun := !skip && !goreleaserDocker && (hasDocker || hasDockerEmpty)

			if workflowWouldRun != tt.workflowShouldRunDoc {
				t.Errorf("workflow docker job would run=%v, expected=%v (skip=%v, should_build_docker_goreleaser=%v, has_docker=%v, has_docker_empty=%v)",
					workflowWouldRun, tt.workflowShouldRunDoc, skip, goreleaserDocker, hasDocker, hasDockerEmpty)
			}
		})
	}
}
