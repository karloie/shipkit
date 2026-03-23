package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// parseFlagsOrExit parses flags and exits on error
func parseFlagsOrExit(fs *flag.FlagSet, args []string) {
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
}

// loadPlanOrWarn loads plan.json and warns if it fails (non-fatal)
func loadPlanOrWarn(path string) *Plan {
	plan, err := loadPlan(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
		return nil
	}
	if plan != nil {
		fmt.Fprintf(os.Stderr, "📋 Loaded: %s\n", path)
	}
	return plan
}

// getEnvOrDefault gets environment variable with fallback
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// writePlanJSON writes plan data to file
func writePlanJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}
