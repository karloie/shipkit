package main

import (
	"fmt"
	"os"
	"strings"
)

// GenerateMakeflowMermaid generates a Mermaid diagram showing the Make target dependency graph
func GenerateMakeflowMermaid(graph *MakeGraph, target string, completed map[string]string) string {
	var buf strings.Builder

	buf.WriteString("```mermaid\n")
	buf.WriteString("graph TD\n")

	// Get all targets involved in building this target
	tree := graph.GetDependencyTree(target)

	if len(tree) == 0 {
		buf.WriteString("```\n")
		return buf.String()
	}

	// Add nodes with status-based styling
	for _, name := range tree {
		status := completed[name]
		if status == "" {
			status = "pending"
		}

		emoji := getTargetEmoji(name)
		safeName := makeSafeMermaidName(name)
		buf.WriteString(fmt.Sprintf("    %s[\"%s %s\"]:::%s\n",
			safeName, emoji, name, status))
	}

	buf.WriteString("\n")

	// Add edges (dependencies)
	for _, name := range tree {
		t := graph.Targets[name]
		if t != nil {
			for _, dep := range t.Dependencies {
				// Only add edge if dependency is in our tree
				if contains(tree, dep) {
					buf.WriteString(fmt.Sprintf("    %s --> %s\n",
						makeSafeMermaidName(dep), makeSafeMermaidName(name)))
				}
			}
		}
	}

	// Add style definitions
	buf.WriteString("\n")
	buf.WriteString("    classDef pending fill:#0366d6,stroke:#0366d6,color:#fff\n")
	buf.WriteString("    classDef running fill:#0969da,stroke:#0969da,color:#fff\n")
	buf.WriteString("    classDef success fill:#1a7f37,stroke:#1a7f37,color:#fff\n")
	buf.WriteString("    classDef failure fill:#cf222e,stroke:#cf222e,color:#fff\n")
	buf.WriteString("    classDef skipped fill:#6e7781,stroke:#6e7781,color:#fff\n")

	buf.WriteString("```\n")

	return buf.String()
}

// getTargetEmoji returns an appropriate emoji for common Make target names
func getTargetEmoji(target string) string {
	// Check for ci- prefixed targets first
	if after, ok := strings.CutPrefix(target, "ci-"); ok {
		target = after
	}

	switch target {
	case "build", "compile":
		return "🔨"
	case "test", "tests":
		return "🧪"
	case "clean":
		return "🧹"
	case "generate", "gen", "codegen":
		return "⚙️"
	case "publish", "deploy", "release":
		return "📦"
	case "install", "deps", "dependencies":
		return "📥"
	case "lint", "format", "fmt":
		return "✨"
	case "verify", "check":
		return "✅"
	case "package":
		return "📦"
	case "docker":
		return "🐳"
	case "frontend":
		return "🎨"
	case "backend":
		return "⚙️"
	default:
		return "📋"
	}
}

// makeSafeMermaidName converts a target name to a valid Mermaid node ID
func makeSafeMermaidName(name string) string {
	// Replace characters that might cause issues in Mermaid
	safe := strings.ReplaceAll(name, "-", "_")
	safe = strings.ReplaceAll(safe, ".", "_")
	safe = strings.ReplaceAll(safe, "/", "_")
	safe = strings.ReplaceAll(safe, ":", "_")
	return safe
}

// contains checks if a string slice contains a value
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

// WriteMermaidToSummary writes a Mermaid diagram to GitHub Actions step summary
func WriteMermaidToSummary(title string, mermaid string) error {
	summaryFile := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryFile == "" {
		// Not in GitHub Actions, skip
		return nil
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("## %s\n\n", title))
	buf.WriteString(mermaid)
	buf.WriteString("\n")

	// Append to summary file
	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(buf.String())
	return err
}

// UpdateMermaidInSummary replaces the entire summary with updated Mermaid
// (GitHub Actions doesn't support partial updates, so we overwrite)
func UpdateMermaidInSummary(title string, mermaid string) error {
	summaryFile := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryFile == "" {
		return nil
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("## %s\n\n", title))
	buf.WriteString(mermaid)
	buf.WriteString("\n")

	return os.WriteFile(summaryFile, []byte(buf.String()), 0644)
}
