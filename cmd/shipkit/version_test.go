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

func TestComputeVersionAutoIncrementsOnExistingTag(t *testing.T) {
	git := &GitProviderMock{LatestTag: "v1.0.0", ExistsTags: map[string]bool{"v1.0.1": true}}
	_, next, _, err := computeVersion("workflow_dispatch", "patch", git, &PRProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != "v1.0.2" {
		t.Errorf("expected v1.0.2, got %s", next)
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
		t.Errorf("expected v0.0.1 (first release with no tags), got %s", next)
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
				if publish != ReleaseSkip {
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
		{"feat with scope", "abc123 feat(api): new endpoint", "v1.1.0"},
		{"fix commit", "abc123 fix: bug fix", "v1.0.1"},
		{"fix with scope", "abc123 fix(auth): wrong token", "v1.0.1"},
		{"no release commit", "abc123 chore: refactor", ""},
		{"fixing word does not trigger", "abc123 fixing badge for homebrew", ""},
		{"featuring word does not trigger", "abc123 featuring new design", ""},
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
				if publish != ReleaseSkip {
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

func TestComputeVersionBumpTypes(t *testing.T) {
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
