package main

import (
	"os"
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
		{"found go", "Go", true, "Go"},
		{"found nodejs", "Node", true, "Node"},
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

func TestNodeJSDetectionWithPackageJson(t *testing.T) {
	// With package.json
	packageJSON := `{
		"name": "test-client-project",
		"version": "1.0.0",
		"description": "Test client application with Node.js"
	}`

	// Create test file
	if err := os.WriteFile("package.json", []byte(packageJSON), 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}
	defer os.Remove("package.json")

	// Check description
	description := detectProjectDescription()
	if description != "Test client application with Node.js" {
		t.Errorf("detectProjectDescription() = %q, want %q", description, "Test client application with Node.js")
	}

	// Check Node.js
	detected := detectProjectTypesWithLogging(false)
	hasNodeJS := false
	for _, p := range detected {
		if p.Name == "Node" {
			hasNodeJS = true
			break
		}
	}

	if !hasNodeJS {
		t.Error("Expected Node.js to be detected when package.json exists")
	}
}

func TestNodeJSDetectionWithoutPackageJson(t *testing.T) {
	// Without package.json
	os.Remove("package.json")

	// Check description
	description := detectProjectDescription()
	if description != "" {
		t.Errorf("detectProjectDescription() = %q, want empty string when no package.json", description)
	}

	// Check not detected
	detected := detectProjectTypesWithLogging(false)
	hasNodeJS := false
	for _, p := range detected {
		if p.Name == "Node" {
			hasNodeJS = true
			break
		}
	}

	// Validate behavior
	if hasNodeJS && fileExists("package.json") {
		t.Log("Note: Node.js detected because package.json exists in project")
	} else if hasNodeJS && !fileExists("package.json") {
		t.Error("Node.js should NOT be detected when package.json doesn't exist")
	}
}

func TestNodeJSDetectionInvalidPackageJson(t *testing.T) {
	// Invalid JSON
	invalidJSON := `{invalid json}`

	if err := os.WriteFile("package.json", []byte(invalidJSON), 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}
	defer os.Remove("package.json")

	// Empty on invalid
	description := detectProjectDescription()
	if description != "" {
		t.Errorf("detectProjectDescription() should return empty string for invalid JSON, got %q", description)
	}

	// Still detected
	detected := detectProjectTypesWithLogging(false)
	hasNodeJS := false
	for _, p := range detected {
		if p.Name == "Node" {
			hasNodeJS = true
			break
		}
	}

	if !hasNodeJS {
		t.Error("Node.js should be detected when package.json exists, even if invalid")
	}
}

func TestNodeJSDetectionEmptyDescription(t *testing.T) {
	// No description field
	packageJSON := `{
		"name": "test-project",
		"version": "1.0.0"
	}`

	if err := os.WriteFile("package.json", []byte(packageJSON), 0644); err != nil {
		t.Fatalf("failed to create package.json: %v", err)
	}
	defer os.Remove("package.json")

	// Empty when missing
	description := detectProjectDescription()
	if description != "" {
		t.Errorf("detectProjectDescription() should return empty string when description missing, got %q", description)
	}

	// Still detected
	detected := detectProjectTypesWithLogging(false)
	hasNodeJS := false
	for _, p := range detected {
		if p.Name == "Node" {
			hasNodeJS = true
			break
		}
	}

	if !hasNodeJS {
		t.Error("Node.js should be detected even when description is missing from package.json")
	}
}
