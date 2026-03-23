package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runPublish executes the publish command using Make
func runPublish(args []string) error {
	fs := newFlagSet("publish")

	target := fs.String("target", "publish", "Make target")
	makefile := fs.String("makefile", "Makefile", "Makefile path")
	visualize := fs.Bool("visualize", true, "Generate Mermaid diagram")
	parseFlagsOrExit(fs, args)

	// Log inputs
	logInputs(map[string]string{
		"target":    *target,
		"makefile":  *makefile,
		"visualize": fmt.Sprintf("%v", *visualize),
	})

	// Determine which target to use (ci- prefix or regular)
	actualTarget, err := selectPublishTarget(*makefile, *target)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "🚀 Publishing with target: %s\n", actualTarget)

	// Parse Makefile for visualization
	var graph *MakeGraph
	if *visualize {
		graph, err = ParseMakefile(*makefile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not parse Makefile for visualization: %v\n", err)
			*visualize = false
		}
	}

	// Show initial publish plan
	if *visualize && graph != nil {
		completed := make(map[string]string)
		mermaid := GenerateMakeflowMermaid(graph, actualTarget, completed)

		if err := WriteMermaidToSummary("🚀 Publish Plan", mermaid); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not write visualization: %v\n", err)
		}

		// Also print to console
		fmt.Println("::group::Publish Plan")
		fmt.Println(mermaid)
		fmt.Println("::endgroup::")
	}

	// Execute make with progress tracking
	fmt.Println("::group::Publish Execution")

	cmd := exec.Command("make", actualTarget)
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
		return fmt.Errorf("failed to start make: %w", err)
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
			WriteMermaidToSummary("🚀 Publish Status (Failed)", mermaid)
		}
		return fmt.Errorf("make %s failed: %w", actualTarget, err)
	}

	// Write final visualization on success
	if *visualize && graph != nil {
		mermaid := GenerateMakeflowMermaid(graph, actualTarget, tracker.Completed)
		if err := WriteMermaidToSummary("🚀 Publish Status (Success)", mermaid); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not write final visualization: %v\n", err)
		}
	}

	fmt.Fprintf(os.Stderr, "✅ Publish completed successfully\n")

	// Log outputs
	logOutputs(map[string]string{
		"status": "success",
		"target": actualTarget,
	})

	return nil
}

// selectPublishTarget chooses between ci-publish and publish targets
func selectPublishTarget(makefilePath, preferredTarget string) (string, error) {
	// If preferred target already has ci- prefix, use it
	if strings.HasPrefix(preferredTarget, "ci-") {
		return preferredTarget, nil
	}

	// Try ci- prefix first
	ciTarget := "ci-" + preferredTarget
	graph, err := ParseMakefile(makefilePath)
	if err != nil {
		// If we can't parse Makefile, fallback to trying ci-publish
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
