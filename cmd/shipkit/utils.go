package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// getEnvOrDefault returns the environment variable value or a default if empty
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

// getRequiredEnv returns an environment variable or an error if it doesn't exist
func getRequiredEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("%s environment variable is required", key)
	}
	return val, nil
}

// fileExists checks if a file exists and is not a directory
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// getSecretWithFallbacks tries multiple environment variables for secrets
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

// newFlagSet creates a FlagSet with stderr output
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

// parseCSV splits a comma-separated string and trims spaces
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

// detectDockerfileForWorkflow detects which dockerfile to use for docker workflow.
// Prefers Containerfile over Dockerfile.
func detectDockerfileForWorkflow() string {
	if fileExists(FileContainerfile) {
		return FileContainerfile
	}
	if fileExists(FileDockerfile) {
		return FileDockerfile
	}
	return FileContainerfile // default fallback
}
