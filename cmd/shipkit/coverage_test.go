package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── utils ────────────────────────────────────────────────────────────────────

func TestGetSecretWithFallbacks(t *testing.T) {
	t.Setenv("KEY_A", "")
	t.Setenv("KEY_B", "found")
	t.Setenv("KEY_C", "other")
	if got := getSecretWithFallbacks("KEY_A", "KEY_B", "KEY_C"); got != "found" {
		t.Errorf("got %q, want %q", got, "found")
	}
	if got := getSecretWithFallbacks("KEY_A"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestWriteOutputToFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "out")
	writeOutput(f, "mykey", "myvalue")
	b, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "mykey=myvalue") {
		t.Errorf("output file missing key=value, got: %s", string(b))
	}
}

func TestWriteOutputMultilineUsesHeredoc(t *testing.T) {
	f := filepath.Join(t.TempDir(), "out")
	writeOutput(f, "msg", "line1\nline2")
	b, _ := os.ReadFile(f)
	if !strings.Contains(string(b), "msg<<EOF") {
		t.Errorf("multiline should use heredoc, got: %s", string(b))
	}
}

func TestWriteOutputNoFile(t *testing.T) {
	// Should not panic when no output file is set
	writeOutput("", "key", "val")
}

// ── detect ───────────────────────────────────────────────────────────────────

func TestDetectDockerFilesGoreleaser(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)

	// Neither file present
	found, name := detectDockerFiles("goreleaser")
	if found {
		t.Errorf("expected not found, got %q", name)
	}

	// Containerfile takes priority
	os.WriteFile("Containerfile", []byte("FROM scratch"), 0644)
	found, name = detectDockerFiles("goreleaser")
	if !found || name != "Containerfile" {
		t.Errorf("expected Containerfile, got found=%v name=%q", found, name)
	}

	// Dockerfile as fallback
	os.Remove("Containerfile")
	os.WriteFile("Dockerfile", []byte("FROM scratch"), 0644)
	found, name = detectDockerFiles("goreleaser")
	if !found || name != "Dockerfile" {
		t.Errorf("expected Dockerfile, got found=%v name=%q", found, name)
	}
}

func TestDetectDockerFilesStandalone(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)

	found, _ := detectDockerFiles("docker")
	if found {
		t.Error("expected no docker file")
	}

	os.WriteFile("Containerfile", []byte("FROM scratch"), 0644)
	found, name := detectDockerFiles("docker")
	if !found || name != "Containerfile" {
		t.Errorf("expected Containerfile, got found=%v name=%q", found, name)
	}
}

func TestDetectDockerfileForWorkflow(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)

	// No files → return empty string (no fallback)
	got := detectDockerfileForWorkflow()
	if got != "" {
		t.Errorf("expected empty string when no docker files exist, got %q", got)
	}

	// Containerfile present → returned directly
	os.WriteFile("Containerfile", []byte("FROM scratch"), 0644)
	got = detectDockerfileForWorkflow()
	if got != "Containerfile" {
		t.Errorf("expected Containerfile, got %q", got)
	}
}

func TestDetectProjectNameFromGoMod(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)

	os.WriteFile("go.mod", []byte("module github.com/acme/myapp\n\ngo 1.22\n"), 0644)
	if got := detectProjectName(); got != "myapp" {
		t.Errorf("expected myapp, got %q", got)
	}
}

func TestDetectProjectDescriptionFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)

	os.WriteFile("package.json", []byte(`{"description":"My cool app"}`), 0644)
	if got := detectProjectDescription(); got != "My cool app" {
		t.Errorf("expected 'My cool app', got %q", got)
	}
}

// ── diagram ──────────────────────────────────────────────────────────────────

func TestPrintReleaseDiagramDryRun(t *testing.T) {
	// Should not panic; just exercises the dry-run branch
	printReleaseDiagram(ModeRelease, "v1.0.0", "v1.0.1", true, false, false)
}

func TestPrintReleaseDiagramAllModes(t *testing.T) {
	cases := []struct {
		mode                string
		hasGoreleaserDocker bool
	}{
		{ModeRelease, false},
		{ModeRelease, true},
		{ModeRerelease, false},
		{ModeDocker, false},
		{ModeGoreleaser, false},
	}
	for _, c := range cases {
		printReleaseDiagram(c.mode, "v1.0.0", "v1.0.1", false, c.hasGoreleaserDocker, false)
	}
	// tag-only (no latest)
	printReleaseDiagram(ModeRerelease, "", "v1.0.1", false, false, false)
}

// ── policy ───────────────────────────────────────────────────────────────────

func TestBuildTagModeSummaryDockerWithSHA(t *testing.T) {
	got := buildTagModeSummary(ModeDocker, "v1.2.3", "1.2.3", "1.2", "owner/img", "abc123", PublishTrue)
	if !strings.Contains(got, "sha-abc123") {
		t.Errorf("expected sha tag, got: %s", got)
	}
	if !strings.Contains(got, "owner/img:1.2.3") {
		t.Errorf("expected version tag, got: %s", got)
	}
}

func TestBuildTagModeSummaryDockerDryRun(t *testing.T) {
	got := buildTagModeSummary(ModeDocker, "v1.2.3", "1.2.3", "1.2", "owner/img", "", PublishFalse)
	if strings.Contains(got, "Image tags") {
		t.Errorf("dry-run should not list image tags, got: %s", got)
	}
}

func TestBuildTagModeSummaryGoreleaser(t *testing.T) {
	got := buildTagModeSummary(ModeGoreleaser, "v1.2.3", "1.2.3", "1.2", "", "", PublishTrue)
	if !strings.Contains(got, "v1.2.3") {
		t.Errorf("expected tag in summary, got: %s", got)
	}
}

func TestBuildSummaryReleaseWithSHA(t *testing.T) {
	got := buildSummary(ModeRelease, "v1.0.0", "v1.0.1", "owner/img", "1.0.1", "def456")
	if !strings.Contains(got, "sha-def456") {
		t.Errorf("expected sha tag, got: %s", got)
	}
}

func TestComputeTagBasedPolicyMissingTag(t *testing.T) {
	_, err := computeTagBasedPolicy(PolicyInput{
		Mode:          ModeDocker,
		ResolveLatest: false,
	}, &EnvProviderMock{}, &GitProviderMock{})
	if err == nil || !strings.Contains(err.Error(), "next-tag is required") {
		t.Errorf("expected next-tag error, got: %v", err)
	}
}

func TestComputeTagBasedPolicyResolveLatestError(t *testing.T) {
	_, err := computeTagBasedPolicy(PolicyInput{
		Mode:          ModeDocker,
		ResolveLatest: true,
	}, &EnvProviderMock{}, &GitProviderMock{Err: os.ErrNotExist})
	if err == nil {
		t.Error("expected error when git fails to resolve tag")
	}
}

func TestComputeReleasePolicyRereleaseResolveGitError(t *testing.T) {
	_, err := computeReleasePolicy(PolicyInput{
		Mode:          ModeRerelease,
		Publish:       PublishTrue,
		ResolveLatest: true,
	}, &EnvProviderMock{}, &GitProviderMock{Err: os.ErrNotExist})
	if err == nil {
		t.Error("expected error when git fails during rerelease resolve")
	}
}

// ── version ──────────────────────────────────────────────────────────────────

func TestComputeVersionPushWithCommitMarker(t *testing.T) {
	// commit log must match regex: ^[a-f0-9]+ fix
	git := &GitProviderMock{
		LatestTag:  "v1.2.3",
		CommitLog:  "abc1234 fix: something important",
		ExistsTags: map[string]bool{"v1.2.4": false},
	}
	pr := &PRProviderMock{Labels: ""}

	_, next, publish, err := computeVersion("push", "", git, pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publish != PublishTrue || next != "v1.2.4" {
		t.Errorf("expected publish=true next=v1.2.4, got publish=%s next=%s", publish, next)
	}
}

func TestComputeVersionInvalidTagFormat(t *testing.T) {
	git := &GitProviderMock{LatestTag: "not-a-semver"}
	_, _, _, err := computeVersion("workflow_dispatch", "patch", git, &PRProviderMock{})
	if err == nil {
		t.Error("expected error for invalid tag format")
	}
}
