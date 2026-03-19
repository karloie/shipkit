package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func globExists(pattern string) bool {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false
	}
	return len(matches) > 0
}

func getSecretWithFallbacks(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}

// parseRepoFormat splits "owner/repo" format and validates it
func parseRepoFormat(repo string) (owner, name string, err error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repo must be in format: owner/name")
	}
	return parts[0], parts[1], nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func parseCSV(v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}

	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// shortenSHA returns the first 7 characters of a git SHA
func shortenSHA(sha string) string {
	s := strings.TrimSpace(sha)
	if len(s) >= 7 {
		return s[:7]
	}
	return s
}

func detectDockerFiles(fileType string) (bool, string) {
	if fileType == "goreleaser" {
		if fileExists(FileContainerfileGoreleaser) {
			return true, FileContainerfileGoreleaser
		}
		if fileExists(FileDockerfileGoreleaser) {
			return true, FileDockerfileGoreleaser
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
	return FileContainerfile // default fallback
}

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
	{"🐹", "Go", FileGo, false},
	{"📦", "Node.js", FilePackageJSON, false},
	{"☕", "Maven", FileMaven, false},
	{"🐘", "Gradle", FileGradle, false},
	{"🐘", "Gradle", FileGradleKts, false},
	{"🍃", "Spring Boot", FileApplicationProps + ",src/main/resources/" + FileApplicationProps + "," + FileApplicationYml + ",src/main/resources/" + FileApplicationYml + "," + FileApplicationYaml + ",src/main/resources/" + FileApplicationYaml, false},
	{"🐳", "Docker", FileContainerfile + "," + FileDockerfile, false},
	{"🚀", "GoReleaser", FileGoReleaser + ",.goreleaser.yaml", false},
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
				fmt.Fprintf(os.Stderr, "%s Detected %s project\n", p.Emoji, p.Name)
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
