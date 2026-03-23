package main

import (
	"bufio"
	"os"
	"strings"
)

// MakeTarget represents a target in a Makefile
type MakeTarget struct {
	Name         string
	Dependencies []string
	Commands     []string
}

// MakeGraph represents the parsed Makefile dependency graph
type MakeGraph struct {
	Targets map[string]*MakeTarget
}

// ParseMakefile parses a Makefile and extracts target dependencies
func ParseMakefile(path string) (*MakeGraph, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	graph := &MakeGraph{Targets: make(map[string]*MakeTarget)}
	scanner := bufio.NewScanner(file)

	var currentTarget *MakeTarget

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check if this is a target line (not indented and contains ':')
		if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") && strings.Contains(line, ":") {
			// Handle target lines like "target: dep1 dep2"
			parts := strings.SplitN(line, ":", 2)

			// Skip special targets (variables, conditionals)
			name := strings.TrimSpace(parts[0])
			if strings.Contains(name, "=") || strings.Contains(name, "?") || strings.Contains(name, "%") {
				continue
			}

			target := &MakeTarget{
				Name:     name,
				Commands: []string{},
			}

			// Parse dependencies if present
			if len(parts) > 1 {
				depsStr := strings.TrimSpace(parts[1])
				// Remove inline commands (after ';')
				if idx := strings.Index(depsStr, ";"); idx != -1 {
					depsStr = strings.TrimSpace(depsStr[:idx])
				}
				if depsStr != "" {
					target.Dependencies = strings.Fields(depsStr)
				}
			}

			graph.Targets[name] = target
			currentTarget = target
		} else if currentTarget != nil && (strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ")) {
			// Command line (indented)
			cmd := strings.TrimSpace(line)
			if cmd != "" {
				currentTarget.Commands = append(currentTarget.Commands, cmd)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return graph, nil
}

// GetDependencyTree returns a flattened list of all targets needed to build the given target
// in execution order (dependencies before targets that depend on them)
func (g *MakeGraph) GetDependencyTree(target string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		t, exists := g.Targets[name]
		if !exists {
			return
		}

		// Visit dependencies first (depth-first)
		for _, dep := range t.Dependencies {
			visit(dep)
		}

		result = append(result, name)
	}

	visit(target)
	return result
}

// HasTarget returns true if the Makefile contains the given target
func (g *MakeGraph) HasTarget(target string) bool {
	_, exists := g.Targets[target]
	return exists
}

// GetTargets returns all target names
func (g *MakeGraph) GetTargets() []string {
	targets := make([]string, 0, len(g.Targets))
	for name := range g.Targets {
		targets = append(targets, name)
	}
	return targets
}
