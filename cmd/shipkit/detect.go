package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

type projectPattern struct {
	Emoji    string // Emoji icon for the project type
	Name     string // Project type name
	Patterns string // Comma-separated patterns to check (OR logic)
	IsGlob   bool   // Whether patterns use glob matching
}

// Cached detection results
var (
	detectedProjects []projectPattern
	detectOnce       sync.Once
)

// Project detection patterns in priority order
var projectPatterns = []projectPattern{
	{"🐳", "Docker", FileContainerfile + "," + FileDockerfile, false},
	{"🚀", "GoReleaser", FileGoReleaser + ",.goreleaser.yaml", false},
	{"🚀🐳", "GoReleaser Docker", FileContainerfile + "," + FileDockerfile, false},
}

func detectProjectTypes() []projectPattern {
	detectOnce.Do(func() {
		detectedProjects = detectProjectTypesWithLogging(true)
	})
	return detectedProjects
}

func detectProjectTypesWithLogging(log bool) []projectPattern {
	var detected []projectPattern
	for _, p := range projectPatterns {
		if matchesPattern(p) {
			if log {
				fmt.Fprintf(os.Stderr, "  %s Detected %s project\n", p.Emoji, p.Name)
			}
			detected = append(detected, p)
		}
	}
	if log && len(detected) == 0 {
		fmt.Fprintln(os.Stderr, "  No project build files detected")
	}
	return detected
}

func matchesPattern(p projectPattern) bool {
	for _, pattern := range strings.Split(p.Patterns, ",") {
		pattern = strings.TrimSpace(pattern)
		if p.IsGlob && globExists(pattern) {
			return true
		}
		if !p.IsGlob && fileExists(pattern) {
			return true
		}
	}
	return false
}

func hasProjectType(detected []projectPattern, name string) bool {
	for _, p := range detected {
		if p.Name == name {
			return true
		}
	}
	return false
}

func getProjectPattern(name string) (projectPattern, bool) {
	for _, p := range projectPatterns {
		if p.Name == name {
			return p, true
		}
	}
	return projectPattern{}, false
}

func detectDockerFiles(fileType string) (bool, string) {
	if fileType == "goreleaser" {
		if fileExists(FileContainerfile) {
			return true, FileContainerfile
		}
		if fileExists(FileDockerfile) {
			return true, FileDockerfile
		}
		return false, ""
	}
	if fileExists(FileContainerfile) {
		return true, FileContainerfile
	}
	if fileExists(FileDockerfile) {
		return true, FileDockerfile
	}
	return false, ""
}

func detectDockerfileForWorkflow() string {
	hasDocker, dockerFile := detectDockerFiles("workflow")
	if hasDocker {
		return dockerFile
	}
	return "" // No docker file found - don't return a default
}

func detectProjectName() string {
	if !fileExists("go.mod") {
		return ""
	}

	data, err := os.ReadFile("go.mod")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			modulePath := strings.TrimSpace(strings.TrimPrefix(line, "module"))
			// Extract last component of module path
			parts := strings.Split(modulePath, "/")
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
	}
	return ""
}

func detectProjectDescription() string {
	if !fileExists("package.json") {
		return ""
	}

	data, err := os.ReadFile("package.json")
	if err != nil {
		return ""
	}

	var pkg struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &pkg); err == nil {
		return pkg.Description
	}
	return ""
}
