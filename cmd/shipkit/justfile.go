package main

import (
	"bufio"
	"os"
	"strings"
)

// JustRecipe represents a recipe in a justfile
type JustRecipe struct {
	Name         string
	Dependencies []string
	Commands     []string
}

// JustGraph represents the parsed justfile dependency graph
type JustGraph struct {
	Recipes map[string]*JustRecipe
}

// ParseJustfile parses a justfile and extracts recipe dependencies
func ParseJustfile(path string) (*JustGraph, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	graph := &JustGraph{Recipes: make(map[string]*JustRecipe)}
	scanner := bufio.NewScanner(file)

	var currentRecipe *JustRecipe

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check if this is a recipe line (not indented and contains ':')
		if !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") && strings.Contains(line, ":") {
			// Handle recipe lines like "recipe: dep1 dep2"
			parts := strings.SplitN(line, ":", 2)

			// Skip variable assignments
			name := strings.TrimSpace(parts[0])
			if strings.Contains(name, "=") || strings.Contains(name, ":=") {
				continue
			}

			recipe := &JustRecipe{
				Name:     name,
				Commands: []string{},
			}

			// Parse dependencies if present
			if len(parts) > 1 {
				depsStr := strings.TrimSpace(parts[1])
				// Remove inline comments (after '#')
				if idx := strings.Index(depsStr, "#"); idx != -1 {
					depsStr = strings.TrimSpace(depsStr[:idx])
				}
				if depsStr != "" {
					recipe.Dependencies = strings.Fields(depsStr)
				}
			}

			graph.Recipes[name] = recipe
			currentRecipe = recipe
		} else if currentRecipe != nil && (strings.HasPrefix(line, "\t") || strings.HasPrefix(line, "    ")) {
			// Command line (indented)
			cmd := strings.TrimSpace(line)
			// Remove @ prefix (just suppresses echo in just)
			cmd = strings.TrimPrefix(cmd, "@")
			cmd = strings.TrimSpace(cmd)
			if cmd != "" {
				currentRecipe.Commands = append(currentRecipe.Commands, cmd)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return graph, nil
}

// GetDependencyTree returns a flattened list of all recipes needed to build the given recipe
// in execution order (dependencies before recipes that depend on them)
func (g *JustGraph) GetDependencyTree(recipe string) []string {
	visited := make(map[string]bool)
	var result []string

	var visit func(string)
	visit = func(name string) {
		if visited[name] {
			return
		}
		visited[name] = true

		r, exists := g.Recipes[name]
		if !exists {
			return
		}

		// Visit dependencies first (depth-first)
		for _, dep := range r.Dependencies {
			visit(dep)
		}

		result = append(result, name)
	}

	visit(recipe)
	return result
}

// HasRecipe returns true if the justfile contains the given recipe
func (g *JustGraph) HasRecipe(recipe string) bool {
	_, exists := g.Recipes[recipe]
	return exists
}

// GetRecipes returns all recipe names
func (g *JustGraph) GetRecipes() []string {
	recipes := make([]string, 0, len(g.Recipes))
	for name := range g.Recipes {
		recipes = append(recipes, name)
	}
	return recipes
}
