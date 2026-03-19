package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
