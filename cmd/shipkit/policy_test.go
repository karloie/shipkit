package main

import (
	"strings"
	"testing"
)

func TestComputeReleasePolicyReleaseSuccess(t *testing.T) {
	env := &EnvProviderMock{values: map[string]string{"DOCKERHUB_USERNAME": "u", "DOCKERHUB_TOKEN": "t"}}
	p, err := computeReleasePolicy(PolicyInput{
		Mode:            "release",
		Release:         "true",
		LatestTag:       "v1.2.2",
		NextTag:         "v1.2.3",
		Image:           "karloie/kompass",
		SHA:             "abcdef1234",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN"},
	}, env, &GitProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Skip != "false" || p.Version != "1.2.3" || p.ReleaseTag != "v1.2.3" {
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
		Release:         "true",
		ResolveLatest:   true,
		Image:           "karloie/kompass",
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN", "HOMEBREW_TAP_GITHUB_TOKEN"},
	}, env, &GitProviderMock{LatestTag: "v9.8.7"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ReleaseTag != "v9.8.7" || p.Version != "9.8.7" {
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
	// When no Containerfile/Dockerfile exists, should skip
	if p.Skip != "true" {
		t.Fatalf("expected skip=true when no dockerfile exists, got: %+v", p)
	}
	if p.Message != "Info: No Containerfile or Dockerfile found. Docker build will be skipped." {
		t.Fatalf("unexpected message: %s", p.Message)
	}
}

func TestComputeReleasePolicyGoReleaserModes(t *testing.T) {
	p, err := computeReleasePolicy(PolicyInput{Mode: "goreleaser", EventName: "workflow_dispatch", Release: "false", NextTag: "v1.2.3"}, &EnvProviderMock{}, &GitProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Skip != "true" {
		t.Fatalf("unexpected skip policy: %+v", p)
	}

	env := &EnvProviderMock{values: map[string]string{"HOMEBREW_TAP_GITHUB_TOKEN": "h"}}
	p, err = computeReleasePolicy(PolicyInput{Mode: "goreleaser", EventName: "push", Release: "false", NextTag: "v1.2.3", RequiredSecrets: []string{"HOMEBREW_TAP_GITHUB_TOKEN"}}, env, &GitProviderMock{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Skip != "false" {
		t.Fatalf("unexpected push policy: %+v", p)
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
		Release:         ReleaseTrue,
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
		Release:         ReleaseTrue,
		NextTag:         "v1.0.0",
		RequiredSecrets: []string{"MISSING_SECRET"},
	}, &EnvProviderMock{}, &GitProviderMock{})
	if err == nil {
		t.Fatal("expected error for missing secret")
	}
	if !strings.Contains(err.Error(), "missing required secret(s)") {
		t.Errorf("error = %v, want 'missing required secret(s)'", err)
	}
}

func TestComputeTagBasedPolicyResolveLatest(t *testing.T) {
	env := &EnvProviderMock{values: map[string]string{"DOCKERHUB_USERNAME": "u", "DOCKERHUB_TOKEN": "t"}}
	git := &GitProviderMock{LatestTag: "v5.4.3"}

	p, err := computeTagBasedPolicy(PolicyInput{
		Mode:            ModeDocker,
		EventName:       "workflow_dispatch",
		Release:         ReleaseTrue,
		ResolveLatest:   true,
		RequiredSecrets: []string{"DOCKERHUB_USERNAME", "DOCKERHUB_TOKEN"},
	}, env, git)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Version != "5.4.3" || p.ReleaseTag != "v5.4.3" {
		t.Errorf("unexpected policy: %+v", p)
	}
}

func TestValidateRequiredSecrets(t *testing.T) {
	tests := []struct {
		name     string
		required []string
		env      map[string]string
		wantErr  bool
	}{
		{
			name:     "all secrets present",
			required: []string{"SECRET1", "SECRET2"},
			env:      map[string]string{"SECRET1": "val1", "SECRET2": "val2"},
		},
		{
			name:     "missing one secret",
			required: []string{"SECRET1", "SECRET2"},
			env:      map[string]string{"SECRET1": "val1"},
			wantErr:  true,
		},
		{
			name:     "missing multiple secrets",
			required: []string{"SECRET1", "SECRET2", "SECRET3"},
			env:      map[string]string{},
			wantErr:  true,
		},
		{
			name:     "no secrets required",
			required: []string{},
			env:      map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := &EnvProviderMock{values: tt.env}
			err := validateRequiredSecrets(tt.required, env)
			if tt.wantErr && err == nil {
				t.Errorf("validateRequiredSecrets() expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("validateRequiredSecrets() unexpected error: %v", err)
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
		{"push event always true", "push", "", "release", ReleaseTrue, false},
		{"push overrides false input", "push", "false", "goreleaser", ReleaseTrue, false},
		{"workflow_dispatch with true", "workflow_dispatch", "true", "release", ReleaseTrue, false},
		{"workflow_dispatch with false", "workflow_dispatch", "false", "release", ReleaseFalse, false},
		{"workflow_dispatch empty docker mode", "workflow_dispatch", "", ModeDocker, ReleaseTrue, false},
		{"workflow_dispatch empty goreleaser mode", "workflow_dispatch", "", ModeGoreleaser, ReleaseFalse, false},
		{"invalid publish value", "workflow_dispatch", "maybe", "release", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveReleaseMode(tt.eventName, tt.publishInput, tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveReleaseMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resolveReleaseMode() = %v, want %v", got, tt.want)
			}
		})
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
