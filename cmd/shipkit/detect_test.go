package main

import (
	"testing"
)

func TestHasProjectType(t *testing.T) {
	detected := []projectPattern{
		{"🐹", "Go", "go.mod", false},
		{"📦", "Node.js", "package.json", false},
		{"🐳", "Docker", "Containerfile,Dockerfile", false},
	}

	tests := []struct {
		name     string
		detected []projectPattern
		search   string
		want     bool
	}{
		{"found go", detected, "Go", true},
		{"found nodejs", detected, "Node.js", true},
		{"found docker", detected, "Docker", true},
		{"not found", detected, "Maven", false},
		{"empty list", []projectPattern{}, "Go", false},
		{"case sensitive", detected, "go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasProjectType(tt.detected, tt.search)
			if got != tt.want {
				t.Errorf("hasProjectType(%v, %q) = %v, want %v", tt.detected, tt.search, got, tt.want)
			}
		})
	}
}

func TestGetProjectPattern(t *testing.T) {
	tests := []struct {
		name      string
		search    string
		wantFound bool
		wantName  string
	}{
		{"found docker", "Docker", true, "Docker"},
		{"found goreleaser", "GoReleaser", true, "GoReleaser"},
		{"not found", "NonExistent", false, ""},
		{"case sensitive", "go", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := getProjectPattern(tt.search)
			if found != tt.wantFound {
				t.Errorf("getProjectPattern(%q) found = %v, want %v", tt.search, found, tt.wantFound)
			}
			if found && got.Name != tt.wantName {
				t.Errorf("getProjectPattern(%q).Name = %v, want %v", tt.search, got.Name, tt.wantName)
			}
		})
	}
}

func TestDetectProjectTypesWithLogging(t *testing.T) {
	// Test with/without logging
	detected := detectProjectTypesWithLogging(false)
	_ = len(detected)

	detectedWithLog := detectProjectTypesWithLogging(true)
	_ = len(detectedWithLog)

	// Same results
	if len(detected) != len(detectedWithLog) {
		t.Errorf("detectProjectTypesWithLogging returned different counts: log=false got %d, log=true got %d",
			len(detected), len(detectedWithLog))
	}
}

func TestDetectDockerFiles(t *testing.T) {
	tests := []struct {
		name     string
		fileType string
	}{
		{"workflow type", "workflow"},
		{"goreleaser type", "goreleaser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, fileName := detectDockerFiles(tt.fileType)
			// Check return validity
			if found && fileName == "" {
				t.Errorf("detectDockerFiles(%q) returned found=true but empty filename", tt.fileType)
			}
			if !found && fileName != "" {
				t.Errorf("detectDockerFiles(%q) returned found=false but non-empty filename: %s", tt.fileType, fileName)
			}
		})
	}
}

func TestDetectProjectTypesCaching(t *testing.T) {
	// Caching test
	first := detectProjectTypes()
	second := detectProjectTypes()
	third := detectProjectTypes()

	// Same cached results
	if len(first) != len(second) || len(second) != len(third) {
		t.Errorf("detectProjectTypes returned different lengths across calls: %d, %d, %d",
			len(first), len(second), len(third))
	}

	// Check content
	for i := 0; i < len(first); i++ {
		if first[i].Name != second[i].Name || second[i].Name != third[i].Name {
			t.Errorf("detectProjectTypes returned different project names at index %d: %s, %s, %s",
				i, first[i].Name, second[i].Name, third[i].Name)
		}
	}
}
