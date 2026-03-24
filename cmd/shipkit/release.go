package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runRelease executes the publish command using Make/just/task
func runRelease(args []string) error {
	// Log raw args BEFORE parsing
	logInputs(map[string]string{
		"raw_args": strings.Join(args, " "),
	})
	fs := newFlagSet("publish")

	target := fs.String("target", "release", "Make target")
	makefile := fs.String("makefile", "Makefile", "Makefile path")
	visualize := fs.Bool("visualize", true, "Generate Mermaid diagram")
	parseFlagsOrExit(fs, args)

	// Detect build orchestrator from plan.json or fallback to auto-detection
	orchestrator := "make" // default
	orchestratorFile := *makefile

	// Try to load plan.json to get orchestrator
	plan, err := loadPlan("plan.json")
	if err == nil && plan != nil && plan.BuildOrchestrator != "" {
		orchestrator = plan.BuildOrchestrator
		// Update file path based on orchestrator
		switch orchestrator {
		case "just":
			orchestratorFile = "justfile"
		case "task":
			orchestratorFile = "Taskfile.yml"
		case "make":
			orchestratorFile = *makefile
		}
	} else {
		// Fallback: auto-detect orchestrator
		if fileExists("justfile") {
			orchestrator = "just"
			orchestratorFile = "justfile"
		} else if fileExists("Taskfile.yml") || fileExists("Taskfile.yaml") {
			orchestrator = "task"
			orchestratorFile = "Taskfile.yml"
		}
	}

	// Log inputs
	logInputs(map[string]string{
		"target":       *target,
		"orchestrator": orchestrator,
		"file":         orchestratorFile,
		"visualize":    fmt.Sprintf("%v", *visualize),
	})

	// Determine which target to use (ci- prefix or regular)
	actualTarget, err := selectPublishTarget(orchestratorFile, *target)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "🚀 Releasing with target: %s (using %s)\n", actualTarget, orchestrator)

	// Parse orchestrator file for visualization
	var graph *MakeGraph
	if *visualize {
		switch orchestrator {
		case "make":
			graph, err = ParseMakefile(orchestratorFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not parse %s file for visualization: %v\n", orchestrator, err)
				*visualize = false
			}
		case "just":
			// Visualization not yet supported for justfile
			*visualize = false
		default:
			// task not supported yet for visualization
			*visualize = false
		}
	}

	// Show initial publish plan
	if *visualize && graph != nil {
		completed := make(map[string]string)
		mermaid := GenerateMakeflowMermaid(graph, actualTarget, completed)

		if err := WriteMermaidToSummary("🚀 Release Plan", mermaid); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not write visualization: %v\n", err)
		}

		// Also print to console
		fmt.Println("::group::Release Plan")
		fmt.Println(mermaid)
		fmt.Println("::endgroup::")
	}

	// Execute orchestrator with progress tracking
	fmt.Println("::group::Release Execution")

	cmd := exec.Command(orchestrator, actualTarget)
	cmd.Env = os.Environ()

	// Set up stdout/stderr forwarding with progress tracking
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", orchestrator, err)
	}

	// Create progress tracker
	tracker := &ProgressTracker{
		Graph:     graph,
		Target:    actualTarget,
		Completed: make(map[string]string),
		Visualize: *visualize,
	}

	// Forward stdout and stderr in goroutines
	done := make(chan error, 2)

	go func() {
		done <- forwardOutput(stdout, os.Stdout, tracker)
	}()

	go func() {
		done <- forwardOutput(stderr, os.Stderr, tracker)
	}()

	// Wait for output forwarding to complete
	<-done
	<-done

	fmt.Println("::endgroup::")

	// Wait for command to finish
	if err := cmd.Wait(); err != nil {
		// Write final visualization on failure
		if *visualize && graph != nil {
			mermaid := GenerateMakeflowMermaid(graph, actualTarget, tracker.Completed)
			WriteMermaidToSummary("🚀 Release Status (Failed)", mermaid)
		}
		return fmt.Errorf("%s %s failed: %w", orchestrator, actualTarget, err)
	}

	// Write final visualization on success
	if *visualize && graph != nil {
		mermaid := GenerateMakeflowMermaid(graph, actualTarget, tracker.Completed)
		if err := WriteMermaidToSummary("🚀 Release Status (Success)", mermaid); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not write final visualization: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "✅ Release completed successfully\n")

	// Log outputs
	logOutputs(map[string]string{
		"status": "success",
		"target": actualTarget,
	})

	return nil
}

// selectPublishTarget chooses between ci-release and release targets
func selectPublishTarget(orchestratorFile, preferredTarget string) (string, error) {
	// If preferred target already has ci- prefix, use it
	if strings.HasPrefix(preferredTarget, "ci-") {
		return preferredTarget, nil
	}

	// Try ci- prefix first
	ciTarget := "ci-" + preferredTarget

	// Detect file type and parse accordingly
	if strings.Contains(orchestratorFile, "justfile") {
		graph, err := ParseJustfile(orchestratorFile)
		if err != nil {
			// If we can't parse justfile, fallback to trying ci-release
			return ciTarget, nil
		}

		// Check if ci- target exists
		if _, exists := graph.Recipes[ciTarget]; exists {
			return ciTarget, nil
		}

		// Check if regular target exists
		if _, exists := graph.Recipes[preferredTarget]; exists {
			return preferredTarget, nil
		}

		// No target found
		return "", fmt.Errorf("no %s or %s recipe found in justfile", ciTarget, preferredTarget)
	} else {
		// Assume Makefile
		graph, err := ParseMakefile(orchestratorFile)
		if err != nil {
			// If we can't parse Makefile, fallback to trying ci-release
			return ciTarget, nil
		}

		// Check if ci- target exists
		if _, exists := graph.Targets[ciTarget]; exists {
			return ciTarget, nil
		}

		// Check if regular target exists
		if _, exists := graph.Targets[preferredTarget]; exists {
			return preferredTarget, nil
		}

		// No target found
		return "", fmt.Errorf("no %s or %s target found in Makefile", ciTarget, preferredTarget)
	}
}
