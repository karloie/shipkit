package main

import (
	"errors"
	"strings"
	"testing"
)

func TestComputeVersionManualPatch(t *testing.T) {
	git := &GitProviderMock{LatestTag: "v0.0.14", ExistsTags: map[string]bool{"v0.0.15": false}}
	pr := &PRProviderMock{}

	latest, next, publish, err := computeVersion("workflow_dispatch", "patch", git, pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest != "v0.0.14" || next != "v0.0.15" || publish != "true" {
		t.Fatalf("unexpected outputs latest=%s next=%s publish=%s", latest, next, publish)
	}
}

func TestComputeVersionSkipWhenNoMarkers(t *testing.T) {
	git := &GitProviderMock{LatestTag: "v1.0.0", CommitLog: "abc chore: x", ExistsTags: map[string]bool{"v1.0.1": false}}
	pr := &PRProviderMock{}

	latest, next, publish, err := computeVersion("push", "", git, pr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest != "v1.0.0" || next != "" || publish != "skip" {
		t.Fatalf("unexpected outputs latest=%s next=%s publish=%s", latest, next, publish)
	}
}

func TestComputeVersionFailsOnExistingTag(t *testing.T) {
	git := &GitProviderMock{LatestTag: "v1.0.0", ExistsTags: map[string]bool{"v1.0.1": true}}
	_, _, _, err := computeVersion("workflow_dispatch", "patch", git, &PRProviderMock{})
	if err == nil {
		t.Fatalf("expected existing-tag error")
	}
}

func TestComputeVersionNoTagFallback(t *testing.T) {
	git := &GitProviderMock{Err: errors.New("no tags")}
	git.Err = nil
	git.LatestTag = ""
	_, next, _, err := computeVersion("workflow_dispatch", "patch", git, &PRProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != "v0.0.1" {
		t.Fatalf("expected v0.0.1, got %s", next)
	}
}

func TestComputeReleasePolicyReleaseSuccess(t *testing.T) {
	env := &EnvProviderMock{values: map[string]string{"DOCKERHUB_USERNAME": "u", "DOCKERHUB_TOKEN": "t"}}
	p, err := computeReleasePolicy(PolicyInput{
		Mode:            "release",
		Publish:         "true",
		LatestTag:       "v1.2.2",
		NextTag:         "v1.2.3",
		Image:           "karloie/kompass",
		SHA:             "abcdef1234",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN"},
	}, env, &GitProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ShouldPublish || p.DockerVersion != "1.2.3" || p.ReleaseTag != "v1.2.3" {
		t.Fatalf("unexpected policy: %+v", p)
	}
}

func TestComputeReleasePolicyRereleaseResolveLatest(t *testing.T) {
	env := &EnvProviderMock{values: map[string]string{
		"DOCKERHUB_USERNAME":        "u",
		"DOCKERHUB_TOKEN":           "t",
		"HOMEBREW_TAP_GITHUB_TOKEN": "h",
	}}
	p, err := computeReleasePolicy(PolicyInput{
		Mode:            "rerelease",
		Publish:         "true",
		ResolveLatest:   true,
		Image:           "karloie/kompass",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN", "HOMEBREW_TAP_GITHUB_TOKEN"},
	}, env, &GitProviderMock{LatestTag: "v9.8.7"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ReleaseTag != "v9.8.7" || p.DockerVersion != "9.8.7" {
		t.Fatalf("unexpected policy: %+v", p)
	}
}

func TestComputeReleasePolicyDockerMode(t *testing.T) {
	env := &EnvProviderMock{values: map[string]string{"DOCKERHUB_USERNAME": "u", "DOCKERHUB_TOKEN": "t"}}
	p, err := computeReleasePolicy(PolicyInput{
		Mode:            "docker",
		EventName:       "workflow_dispatch",
		NextTag:         "v1.2.3",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN"},
	}, env, &GitProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ShouldPublish || p.PublishMode != "true" || p.DockerMajorMinor != "1.2" {
		t.Fatalf("unexpected policy: %+v", p)
	}
}

func TestComputeReleasePolicyGoReleaserModes(t *testing.T) {
	p, err := computeReleasePolicy(PolicyInput{Mode: "goreleaser", EventName: "workflow_dispatch", Publish: "false", NextTag: "v1.2.3"}, &EnvProviderMock{}, &GitProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.PublishMode != "false" || p.ShouldPublish {
		t.Fatalf("unexpected dry-run policy: %+v", p)
	}

	env := &EnvProviderMock{values: map[string]string{"HOMEBREW_TAP_GITHUB_TOKEN": "h"}}
	p, err = computeReleasePolicy(PolicyInput{Mode: "goreleaser", EventName: "push", Publish: "false", NextTag: "v1.2.3", RequiredSecrets: []string{"HOMEBREW_TAP_GITHUB_TOKEN"}}, env, &GitProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.PublishMode != "true" || !p.ShouldPublish {
		t.Fatalf("unexpected push policy: %+v", p)
	}
}

func TestHelpers(t *testing.T) {
	if v, err := parseTagVersion("v1.2.3"); err != nil || v != "1.2.3" {
		t.Fatalf("unexpected parseTagVersion result: %q %v", v, err)
	}
	if _, err := parseTagVersion("1.2.3"); err == nil {
		t.Fatalf("expected invalid tag error")
	}
	if mm, err := parseMajorMinor("1.2.3"); err != nil || mm != "1.2" {
		t.Fatalf("unexpected parseMajorMinor result: %q %v", mm, err)
	}
	if got := parseCSV("A, B ,,C"); len(got) != 3 {
		t.Fatalf("unexpected parseCSV result: %#v", got)
	}
	if s := buildSummary("rerelease", "", "v1.2.3", "karloie/kompass", "1.2.3", "abcdef1"); !strings.Contains(s, "Re-released tag") {
		t.Fatalf("unexpected rerelease summary: %s", s)
	}
}

func TestValidateRequiredSecrets(t *testing.T) {
	tests := []struct {
		name     string
		required []string
		env      map[string]string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "all secrets present",
			required: []string{"SECRET1", "SECRET2"},
			env:      map[string]string{"SECRET1": "val1", "SECRET2": "val2"},
			wantErr:  false,
		},
		{
			name:     "missing one secret",
			required: []string{"SECRET1", "SECRET2"},
			env:      map[string]string{"SECRET1": "val1"},
			wantErr:  true,
			errMsg:   "missing required secret(s): SECRET2",
		},
		{
			name:     "missing multiple secrets",
			required: []string{"SECRET1", "SECRET2", "SECRET3"},
			env:      map[string]string{},
			wantErr:  true,
			errMsg:   "missing required secret(s): SECRET1, SECRET2, SECRET3",
		},
		{
			name:     "no secrets required",
			required: []string{},
			env:      map[string]string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvProviderMock{values: tt.env}
			err := validateRequiredSecrets(tt.required, env)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequiredSecrets() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("error message = %q, want to contain %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestResolvePublishMode(t *testing.T) {
	tests := []struct {
		name         string
		eventName    string
		publishInput string
		mode         string
		want         string
		wantErr      bool
	}{
		{"push event always true", "push", "", "release", PublishTrue, false},
		{"push overrides false input", "push", "false", "goreleaser", PublishTrue, false},
		{"workflow_dispatch with true", "workflow_dispatch", "true", "release", PublishTrue, false},
		{"workflow_dispatch with false", "workflow_dispatch", "false", "release", PublishFalse, false},
		{"workflow_dispatch empty docker mode", "workflow_dispatch", "", ModeDocker, PublishTrue, false},
		{"workflow_dispatch empty goreleaser mode", "workflow_dispatch", "", ModeGoreleaser, PublishFalse, false},
		{"invalid publish value", "workflow_dispatch", "maybe", "release", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePublishMode(tt.eventName, tt.publishInput, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolvePublishMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolvePublishMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMajorMinor(t *testing.T) {
	tests := []struct {
		version string
		want    string
		wantErr bool
	}{
		{"1.2.3", "1.2", false},
		{"0.0.1", "0.0", false},
		{"10.20.30", "10.20", false},
		{"1.2", "", true},
		{"1", "", true},
		{"1.2.3.4", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := parseMajorMinor(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMajorMinor(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMajorMinor(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseRepoFormat(t *testing.T) {
	tests := []struct {
		repo      string
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{"owner/repo", "owner", "repo", false},
		{"my-org/my-app", "my-org", "my-app", false},
		{"invalid", "", "", true},
		{"/repo", "", "", true},
		{"owner/", "", "", true},
		{"", "", "", true},
		{"owner/repo/extra", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			owner, name, err := parseRepoFormat(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepoFormat(%q) error = %v, wantErr %v", tt.repo, err, tt.wantErr)
				return
			}
			if owner != tt.wantOwner || name != tt.wantName {
				t.Errorf("parseRepoFormat(%q) = (%v, %v), want (%v, %v)", tt.repo, owner, name, tt.wantOwner, tt.wantName)
			}
		})
	}
}

func TestShortenSHA(t *testing.T) {
	tests := []struct {
		sha  string
		want string
	}{
		{"abcdef1234567890", "abcdef1"},
		{"abc", "abc"},
		{"", ""},
		{"  1234567890  ", "1234567"},
		{"short", "short"},
	}

	for _, tt := range tests {
		t.Run(tt.sha, func(t *testing.T) {
			got := shortenSHA(tt.sha)
			if got != tt.want {
				t.Errorf("shortenSHA(%q) = %v, want %v", tt.sha, got, tt.want)
			}
		})
	}
}

func TestParseCSV(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b , c", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
		{"", nil},
		{"  ", nil},
		{"single", []string{"single"}},
		{" a , , b,  c  ", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseCSV(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseCSV(%q) length = %v, want %v", tt.input, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseCSV(%q)[%d] = %v, want %v", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestComputeVersionWithPRLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   string
		wantNext string
		wantBump string
	}{
		{"major label", "release:major\nother", "v2.0.0", "major"},
		{"minor label", "release:minor", "v1.1.0", "minor"},
		{"patch label", "release:patch", "v1.0.1", "patch"},
		{"no release label", "bug\nenhancement", "", "skip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git := &GitProviderMock{
				LatestTag:  "v1.0.0",
				ExistsTags: map[string]bool{tt.wantNext: false},
			}
			pr := &PRProviderMock{Labels: tt.labels}

			_, next, publish, err := computeVersion("push", "", git, pr)
			if err != nil && tt.wantBump != "skip" {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantBump == "skip" {
				if publish != PublishSkip {
					t.Errorf("expected skip, got publish=%s", publish)
				}
			} else {
				if next != tt.wantNext {
					t.Errorf("next = %v, want %v", next, tt.wantNext)
				}
			}
		})
	}
}

func TestComputeVersionWithCommitAnalysis(t *testing.T) {
	tests := []struct {
		name      string
		commitLog string
		wantNext  string
	}{
		{"breaking change", "abc123 feat!: new api", "v2.0.0"},
		{"BREAKING CHANGE footer", "abc123 feat: thing\n\nBREAKING CHANGE: removed api", "v2.0.0"},
		{"feat commit", "abc123 feat: new feature", "v1.1.0"},
		{"fix commit", "abc123 fix: bug fix", "v1.0.1"},
		{"no release commit", "abc123 chore: refactor", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			git := &GitProviderMock{
				LatestTag:  "v1.0.0",
				CommitLog:  tt.commitLog,
				ExistsTags: map[string]bool{tt.wantNext: false},
			}
			pr := &PRProviderMock{}

			_, next, publish, err := computeVersion("push", "", git, pr)
			if err != nil && tt.wantNext != "" {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNext == "" {
				if publish != PublishSkip {
					t.Errorf("expected skip for %s, got publish=%s", tt.name, publish)
				}
			} else {
				if next != tt.wantNext {
					t.Errorf("%s: next = %v, want %v", tt.name, next, tt.wantNext)
				}
			}
		})
	}
}

func TestComputeVersionAllBumpTypes(t *testing.T) {
	tests := []struct {
		bump     string
		wantNext string
	}{
		{BumpMajor, "v2.0.0"},
		{BumpMinor, "v1.1.0"},
		{BumpPatch, "v1.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.bump, func(t *testing.T) {
			git := &GitProviderMock{
				LatestTag:  "v1.0.0",
				ExistsTags: map[string]bool{tt.wantNext: false},
			}

			_, next, _, err := computeVersion("workflow_dispatch", tt.bump, git, &PRProviderMock{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if next != tt.wantNext {
				t.Errorf("next = %v, want %v", next, tt.wantNext)
			}
		})
	}
}

func TestComputeVersionInvalidBump(t *testing.T) {
	git := &GitProviderMock{LatestTag: "v1.0.0"}
	_, _, _, err := computeVersion("workflow_dispatch", "invalid", git, &PRProviderMock{})
	if err == nil {
		t.Fatal("expected error for invalid bump")
	}
	if !strings.Contains(err.Error(), "invalid bump") {
		t.Errorf("error = %v, want 'invalid bump'", err)
	}
}

func TestComputeReleasePolicyInvalidMode(t *testing.T) {
	_, err := computeReleasePolicy(PolicyInput{
		Mode: "invalid-mode",
	}, &EnvProviderMock{}, &GitProviderMock{})
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "invalid mode") {
		t.Errorf("error = %v, want 'invalid mode'", err)
	}
}

func TestComputeReleasePolicyMissingNextTag(t *testing.T) {
	_, err := computeReleasePolicy(PolicyInput{
		Mode:            ModeRelease,
		Publish:         PublishTrue,
		RequiredSecrets: []string{},
	}, &EnvProviderMock{}, &GitProviderMock{})
	if err == nil {
		t.Fatal("expected error for missing next-tag")
	}
	if !strings.Contains(err.Error(), "next-tag is required") {
		t.Errorf("error = %v, want 'next-tag is required'", err)
	}
}

func TestComputeReleasePolicyMissingSecrets(t *testing.T) {
	_, err := computeReleasePolicy(PolicyInput{
		Mode:            ModeRelease,
		Publish:         PublishTrue,
		NextTag:         "v1.0.0",
		RequiredSecrets: []string{"MISSING_SECRET"},
	}, &EnvProviderMock{}, &GitProviderMock{})
	if err == nil {
		t.Fatal("expected error for missing secrets")
	}
	if !strings.Contains(err.Error(), "missing required secret") {
		t.Errorf("error = %v, want 'missing required secret'", err)
	}
}

func TestComputeTagBasedPolicyResolveLatest(t *testing.T) {
	env := &EnvProviderMock{values: map[string]string{"DOCKERHUB_USERNAME": "u", "DOCKERHUB_TOKEN": "t"}}
	git := &GitProviderMock{LatestTag: "v5.4.3"}

	p, err := computeTagBasedPolicy(PolicyInput{
		Mode:            ModeDocker,
		EventName:       "workflow_dispatch",
		Publish:         PublishTrue,
		ResolveLatest:   true,
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN"},
	}, env, git)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.DockerVersion != "5.4.3" || p.ReleaseTag != "v5.4.3" {
		t.Errorf("unexpected policy: %+v", p)
	}
}

func TestBuildSummaryRelease(t *testing.T) {
	summary := buildSummary(ModeRelease, "v1.0.0", "v1.0.1", "owner/repo", "1.0.1", "abc1234")
	if !strings.Contains(summary, "Bumped from: v1.0.0") {
		t.Error("summary should contain bumped from")
	}
	if !strings.Contains(summary, "Released new: v1.0.1") {
		t.Error("summary should contain released new")
	}
	if !strings.Contains(summary, "owner/repo:1.0.1") {
		t.Error("summary should contain image tag")
	}
	if !strings.Contains(summary, "sha-abc1234") {
		t.Error("summary should contain sha tag")
	}
}

func TestParsePRLabels(t *testing.T) {
	tests := []struct {
		labels string
		want   string
	}{
		{"release:major", BumpMajor},
		{"bug\nrelease:minor\nenhancement", BumpMinor},
		{"release:patch", BumpPatch},
		{"bug\nenhancement", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.labels, func(t *testing.T) {
			got := parsePRLabels(tt.labels)
			if got != tt.want {
				t.Errorf("parsePRLabels(%q) = %v, want %v", tt.labels, got, tt.want)
			}
		})
	}
}
