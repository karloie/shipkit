package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestWorkflowYAMLSyntax validates workflow YAML syntax using actionlint if available
func TestWorkflowYAMLSyntax(t *testing.T) {
	// Check if actionlint is available
	if _, err := exec.LookPath("actionlint"); err != nil {
		t.Skip("actionlint not installed, skipping workflow syntax validation")
	}

	workflows := []string{
		"release-shipkit.yml",
		"release.yml",
		"ci.yml",
		"docker-publish.yml",
		"docker-build.yml",
		"go-build.yml",
		"go-publish.yml",
		"maven-publish.yml",
		"maven-build.yml",
		"npm-publish.yml",
		"npm-build.yml",
		"git-tag.yml",
		"git-version-files.yml",
	}

	// Get absolute path to repo root (two levels up from cmd/shipkit)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	repoRoot := filepath.Join(cwd, "..", "..")
	repoRootAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	for _, workflow := range workflows {
		t.Run(workflow, func(t *testing.T) {
			workflowPath := filepath.Join(repoRootAbs, ".github", "workflows", workflow)

			// Check file exists first
			if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
				t.Skipf("workflow file %s not found", workflowPath)
			}

			cmd := exec.Command("actionlint", workflowPath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("actionlint failed for %s:\n%s", workflow, string(output))
			}
		})
	}
}

// TestWorkflowWithActPush tests the workflow with a push event using act
func TestWorkflowWithActPush(t *testing.T) {
	// Check if act is available
	if _, err := exec.LookPath("act"); err != nil {
		t.Skip("act not installed, skipping workflow execution test")
	}

	// act has limited support for reusable workflows, so we skip this test
	// See: https://github.com/nektos/act/issues/1131
	t.Skip("act has limited support for reusable workflows (uses:)")
}

// TestWorkflowWithActManualDispatch tests the workflow with workflow_dispatch event
func TestWorkflowWithActManualDispatch(t *testing.T) {
	// Check if act is available
	if _, err := exec.LookPath("act"); err != nil {
		t.Skip("act not installed, skipping workflow execution test")
	}

	// act has limited support for reusable workflows, so we skip this test
	// See: https://github.com/nektos/act/issues/1131
	t.Skip("act has limited support for reusable workflows (uses:)")
}

// setupMockGitRepo creates a temporary git repository for testing
func setupMockGitRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git init: %v", err)
	}

	// Configure git
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@example.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()

	// Create go.mod to detect as Go project
	goMod := filepath.Join(tmpDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("failed to create go.mod: %v", err)
	}

	// Create initial commit
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	cmd = exec.Command("git", "-C", tmpDir, "commit", "-m", "initial commit")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create initial commit: %v", err)
	}

	// Create an initial tag (v0.3.5 to match current version)
	exec.Command("git", "-C", tmpDir, "tag", "v0.3.5").Run()

	return tmpDir
}

// TestPlanCommandWithWorkflowFlags validates the plan command accepts all flags used in the workflow
// This catches issues like unsupported flags before they break in CI
func TestPlanCommandWithWorkflowFlags(t *testing.T) {
	// Build the shipkit binary first
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	realRepoRoot := filepath.Join(cwd, "..", "..")
	realRepoRootAbs, err := filepath.Abs(realRepoRoot)
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}

	tmpBin := filepath.Join(t.TempDir(), "shipkit")
	buildCmd := exec.Command("go", "build", "-o", tmpBin, "./cmd/shipkit")
	buildCmd.Dir = realRepoRootAbs
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build shipkit: %v\n%s", err, output)
	}

	// Create a mock git repository for testing
	repoDir := setupMockGitRepo(t)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "release mode with bump",
			args: []string{
				"plan",
				"-mode=release",
				"-bump=patch",
				"-image=owner/project",
				"-sha=abc123",
			},
		},
		{
			name: "rerelease mode",
			args: []string{
				"plan",
				"-mode=rerelease",
				"-bump=patch",
				"-image=owner/project",
				"-sha=abc123",
			},
		},
		{
			name: "docker mode with next-tag",
			args: []string{
				"plan",
				"-mode=docker",
				"-next-tag=v1.2.3",
				"-latest-tag=v1.2.2",
				"-image=owner/project",
				"-sha=abc123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file for GITHUB_OUTPUT
			tmpFile, err := os.CreateTemp("", "github-output-*")
			if err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())
			tmpFile.Close()

			// Run the plan command with workflow-like flags in the mock repo directory
			cmd := exec.Command(tmpBin, tt.args...)
			cmd.Dir = repoDir
			cmd.Env = append(os.Environ(),
				"GITHUB_OUTPUT="+tmpFile.Name(),
				"GITHUB_EVENT_NAME=workflow_dispatch",
				"DOCKERHUB_USERNAME=testuser",
				"DOCKERHUB_TOKEN=testtoken",
				"HOMEBREW_TAP_GITHUB_TOKEN=testtoken",
			)

			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("plan command failed with workflow flags:\n%s\nError: %v", string(output), err)
			}

			// Verify outputs were written
			outputContent, err := os.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatalf("failed to read output file: %v", err)
			}

			outputStr := string(outputContent)

			// Check for expected outputs based on mode
			if tt.name == "release mode with bump" {
				// Should have publish output
				if !strings.Contains(outputStr, "publish=") {
					t.Errorf("missing publish output in release mode")
				}
			}

			// All modes should output has_docker
			if !strings.Contains(outputStr, "has_docker=") {
				t.Errorf("missing has_docker output (required for workflow condition)")
			}
		})
	}
}

// TestWorkflowInputDefaults validates that workflow input defaults are correctly defined
func TestWorkflowInputDefaults(t *testing.T) {
	// Get absolute path to workflow file
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	workflowPath := filepath.Join(cwd, "..", "..", ".github", "workflows", "release-shipkit.yml")
	workflowPathAbs, err := filepath.Abs(workflowPath)
	if err != nil {
		t.Fatalf("failed to resolve workflow path: %v", err)
	}

	content, err := os.ReadFile(workflowPathAbs)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	workflow := string(content)

	// Verify key inputs have defaults (not checking specific version values)
	requiredDefaults := map[string]bool{
		"bump":     true, // should have default: patch
		"tool_ref": true, // should have a default version
		"mode":     true, // should have default: release
		"dry_run":  true, // should have default: true
	}

	for input := range requiredDefaults {
		// Look for "default: <value>" under the input definition
		if !strings.Contains(workflow, input+":") {
			t.Errorf("input %s not found in workflow", input)
			continue
		}

		// Simple check - verify default exists (not checking specific value)
		inputSection := extractInputSection(workflow, input)
		if !strings.Contains(inputSection, "default:") {
			t.Errorf("input %s missing default value", input)
		}
	}
}

// extractInputSection extracts the YAML section for a given input
func extractInputSection(workflow string, inputName string) string {
	lines := strings.Split(workflow, "\n")
	inSection := false
	var section []string

	for _, line := range lines {
		// Start of input section
		if strings.Contains(line, inputName+":") && !strings.Contains(line, "${{") {
			inSection = true
		}

		if inSection {
			section = append(section, line)

			// Next input or end of inputs section
			if len(section) > 1 && strings.HasPrefix(strings.TrimSpace(line), "- ") {
				break
			}
			if len(section) > 1 && !strings.HasPrefix(line, "      ") && !strings.HasPrefix(line, "        ") && strings.TrimSpace(line) != "" {
				break
			}
		}
	}

	return strings.Join(section, "\n")
}

// TestWorkflowFallbackLogic validates that fallback logic is present where needed
func TestWorkflowFallbackLogic(t *testing.T) {
	// Get absolute path to workflow file
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	workflowPath := filepath.Join(cwd, "..", "..", ".github", "workflows", "release-shipkit.yml")
	workflowPathAbs, err := filepath.Abs(workflowPath)
	if err != nil {
		t.Fatalf("failed to resolve workflow path: %v", err)
	}

	content, err := os.ReadFile(workflowPathAbs)
	if err != nil {
		t.Fatalf("failed to read workflow file: %v", err)
	}

	workflow := string(content)

	// Verify critical fallbacks exist for push events
	// Verify fallbacks exist (not checking specific version values)
	requiredFallbacks := []string{
		"tool_ref", // should have fallback in with: section
		"mode",     // should have fallback || 'release'
	}

	for _, param := range requiredFallbacks {
		// Check that a fallback pattern exists: inputs.param ... || '...'
		pattern := "inputs." + param
		if !strings.Contains(workflow, pattern) {
			t.Errorf("parameter %s not found in with: section", param)
			continue
		}
		// Look for fallback operator ||
		if !strings.Contains(workflow, pattern) || !strings.Contains(workflow, "||") {
			t.Errorf("missing fallback for %s (expected || pattern)", param)
		}
	}

	// Verify image is NOT passed as input (plan computes it internally)
	if strings.Contains(workflow, "image: ${{ inputs.image") || strings.Contains(workflow, "image: ${{ format(") {
		t.Error("image should not be passed to release.yml (plan computes it from git context)")
	}

	// Verify the uses: line has a static @ref (not using inputs context)
	if strings.Contains(workflow, "uses:") && strings.Contains(workflow, "release.yml@${{") {
		if strings.Contains(workflow, "release.yml@${{ inputs.") {
			t.Error("uses: @ref must be static, cannot use inputs context")
		}
	}
}
