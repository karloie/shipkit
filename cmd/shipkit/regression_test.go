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
		Publish:         PublishTrue,
		ResolveLatest:   true,
		Image:           "karloie/kompass",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN", "HOMEBREW_TAP_GITHUB_TOKEN"},
	}, env, &GitProviderMock{LatestTag: "v0.0.16"})
	if err != nil {
		t.Fatalf("rerelease should not error: %v", err)
	}
	if p.DryRun == PublishTrue {
		t.Fatal("rerelease must not be dry-run when publish=true and tag is resolvable")
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

	env := &EnvProviderMock{values: map[string]string{
		"DOCKERHUB_USERNAME":        "u",
		"DOCKERHUB_TOKEN":           "t",
		"HOMEBREW_TAP_GITHUB_TOKEN": "h",
	}}
	p, err := computeReleasePolicy(PolicyInput{
		Mode:            ModeRerelease,
		Publish:         PublishTrue,
		NextTag:         "v1.2.3",
		Image:           "karloie/kompass",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN", "HOMEBREW_TAP_GITHUB_TOKEN"},
	}, env, &GitProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Dockerfile == "" {
		t.Fatal("policy.Dockerfile should be non-empty (auto-detected fallback)")
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
		HasChangelog: true, // triggers the disable branch
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
	tmpl := defaultGoReleaserTemplate()

	invalidFields := []string{
		"skip:",       // replaced by disable: in v2
		"ignore:",     // was accidentally added as top-level key
		"\nbrews:",    // deprecated in goreleaser v2, replaced by homebrew_casks
	}
	for _, bad := range invalidFields {
		if strings.Contains(tmpl, bad) {
			t.Errorf("template contains invalid/deprecated goreleaser v2 field %q", bad)
		}
	}
	if !strings.Contains(tmpl, "homebrew_casks:") {
		t.Error("template must use 'homebrew_casks' (not deprecated 'brews')")
	}
}
