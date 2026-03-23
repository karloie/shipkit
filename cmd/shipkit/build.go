package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// runBuild executes the build command using Make
func runBuild(args []string) error {
	fs := newFlagSet("build")

	target := fs.String("target", "build", "Make target")
	makefile := fs.String("makefile", "Makefile", "Makefile path")
	visualize := fs.Bool("visualize", true, "Generate Mermaid diagram")
	parseFlagsOrExit(fs, args)

	// Determine which target to use (ci- prefix or regular)
	actualTarget, err := selectBuildTarget(*makefile, *target)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "🔨 Building with target: %s\n", actualTarget)

	// Parse Makefile for visualization
	var graph *MakeGraph
	if *visualize {
		graph, err = ParseMakefile(*makefile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not parse Makefile for visualization: %v\n", err)
			*visualize = false
		}
	}

	// Show initial build plan
	if *visualize && graph != nil {
		completed := make(map[string]string)
		mermaid := GenerateMakeflowMermaid(graph, actualTarget, completed)

		if err := WriteMermaidToSummary("🔨 Build Plan", mermaid); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Could not write visualization: %v\n", err)
		}

		// Also print to console
		fmt.Println("::group::Build Plan")
		fmt.Println(mermaid)
		fmt.Println("::endgroup::")
	}

	// Execute make with progress tracking
	fmt.Println("::group::Build Execution")

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

	// Wait for command to finish
	err = cmd.Wait()

	fmt.Println("::endgroup::")

	// Show final status
	if err != nil {
		if *visualize && graph != nil {
			// Mark as failed
			mermaid := GenerateMakeflowMermaid(graph, actualTarget, tracker.Completed)
			UpdateMermaidInSummary("🔨 Build Failed", mermaid)
		}
		fmt.Fprintf(os.Stderr, "::error::Build failed: %v\n", err)
		return fmt.Errorf("build failed: %w", err)
	}

	if *visualize && graph != nil {
		// Mark all as success
		for _, target := range graph.GetDependencyTree(actualTarget) {
			tracker.Completed[target] = "success"
		}
		mermaid := GenerateMakeflowMermaid(graph, actualTarget, tracker.Completed)
		UpdateMermaidInSummary("🔨 Build Complete ✅", mermaid)
	}

	fmt.Println("::notice::✅ Build completed successfully")
	return nil
}

// selectBuildTarget determines which build target to use (ci-build or build)
func selectBuildTarget(makefile string, requested string) (string, error) {
	graph, err := ParseMakefile(makefile)
	if err != nil {
		// If we can't parse, just try the requested target
		return requested, nil
	}

	// Try ci- prefixed version first
	ciTarget := "ci-" + requested
	if graph.HasTarget(ciTarget) {
		return ciTarget, nil
	}

	// Fall back to unprefixed version
	if graph.HasTarget(requested) {
		return requested, nil
	}

	return "", fmt.Errorf("target '%s' (or '%s') not found in Makefile", requested, ciTarget)
}

// ProgressTracker tracks Make target completion for visualization
type ProgressTracker struct {
	Graph     *MakeGraph
	Target    string
	Completed map[string]string // target -> status (running, success, failure)
	Visualize bool
	buffer    []byte
}

// forwardOutput forwards output from reader to writer while tracking progress
func forwardOutput(reader io.Reader, writer io.Writer, tracker *ProgressTracker) error {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		// Write line to output
		fmt.Fprintln(writer, line)

		// Track progress if visualization is enabled
		if tracker != nil && tracker.Visualize && tracker.Graph != nil {
			tracker.parseLine(line)
		}
	}

	return scanner.Err()
}

// parseLine analyzes Make output to detect target execution
func (pt *ProgressTracker) parseLine(line string) {
	// Make prints lines like:
	// "make[1]: Entering directory..."
	// "make: 'target' is up to date."
	// "make: *** [Makefile:10: target] Error 1"

	// Look for target names in output
	for _, target := range pt.Graph.GetDependencyTree(pt.Target) {
		// Check if target is being executed
		if strings.Contains(line, fmt.Sprintf("make: Nothing to be done for `%s'", target)) ||
			strings.Contains(line, fmt.Sprintf("make: `%s' is up to date", target)) {
			if pt.Completed[target] == "" {
				pt.Completed[target] = "success"
				pt.emitNotice(target, "completed (up to date)")
			}
		}

		// Check for errors
		if strings.Contains(line, fmt.Sprintf("[Makefile:%d: %s]", 0, target)) ||
			strings.Contains(line, fmt.Sprintf("***%s", target)) {
			if pt.Completed[target] != "failure" {
				pt.Completed[target] = "failure"
				pt.emitNotice(target, "FAILED")
			}
		}
	}
}

// emitNotice prints a GitHub Actions notice
func (pt *ProgressTracker) emitNotice(target, status string) {
	fmt.Printf("::notice::Target '%s' %s\n", target, status)
}
