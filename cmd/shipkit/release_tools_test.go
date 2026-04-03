package main

import "testing"

func TestReleaseDockerBuildMetadataUsesPlanShaAndTag(t *testing.T) {
	version, commit, date := releaseDockerBuildMetadata(&Plan{
		TagRelease: "v1.2.3",
		TagNext:    "v1.2.4",
		Sha:        "abc123",
	}, "v9.9.9")

	if version != "v9.9.9" {
		t.Fatalf("expected explicit tag to win, got %q", version)
	}
	if commit != "abc123" {
		t.Fatalf("expected plan sha, got %q", commit)
	}
	if date == "" {
		t.Fatal("expected build date to be populated")
	}
}

func TestReleaseDockerBuildMetadataFallsBackToPlanTagRelease(t *testing.T) {
	version, commit, date := releaseDockerBuildMetadata(&Plan{
		TagRelease: "v1.2.3",
		Sha:        "def456",
	}, "")

	if version != "v1.2.3" {
		t.Fatalf("expected plan release tag, got %q", version)
	}
	if commit != "def456" {
		t.Fatalf("expected plan sha, got %q", commit)
	}
	if date == "" {
		t.Fatal("expected build date to be populated")
	}
}