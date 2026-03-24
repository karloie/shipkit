package main

import (
	"os"
	"testing"
)

func TestParseJustfile_AtPrefixRecipes(t *testing.T) {
	// Create a temporary justfile with @ prefix recipes (to suppress output)
	content := `# Test justfile with @ prefix

IMAGE := "karloie/test"

# Show help
@help:
    echo "Available recipes:"
    just --list

# Build binary
@build:
    go build -o app ./cmd/app

# Run tests
@test: build
    go test ./...

# CI build target
@ci-build:
    go build -o app ./cmd/app

# CI test target  
@ci-test:
    go test ./...

# CI integration test
@ci-integration-test:
    echo "Running integration tests"

# Regular recipe without @ prefix
regular-recipe:
    echo "No @ prefix"
`

	tmpfile, err := os.CreateTemp("", "justfile-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	graph, err := ParseJustfile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseJustfile failed: %v", err)
	}

	// Verify @ prefix is stripped from recipe names
	expectedRecipes := []string{"help", "build", "test", "ci-build", "ci-test", "ci-integration-test", "regular-recipe"}
	for _, name := range expectedRecipes {
		if !graph.HasRecipe(name) {
			t.Errorf("Expected recipe '%s' not found (@ prefix should be stripped)", name)
		}
	}

	// Verify @ prefix is NOT in the recipe names
	recipes := graph.GetRecipes()
	for _, name := range recipes {
		if len(name) > 0 && name[0] == '@' {
			t.Errorf("Recipe name '%s' should not have @ prefix", name)
		}
	}

	// Verify dependencies are parsed correctly
	if recipe, ok := graph.Recipes["test"]; ok {
		if len(recipe.Dependencies) != 1 || recipe.Dependencies[0] != "build" {
			t.Errorf("Expected test to depend on build, got %v", recipe.Dependencies)
		}
	} else {
		t.Error("Recipe 'test' not found")
	}

	// Verify ci-build is detected (critical for shipkit CI hooks)
	if !graph.HasRecipe("ci-build") {
		t.Error("ci-build recipe should be detected (@ prefix stripped)")
	}
	if !graph.HasRecipe("ci-test") {
		t.Error("ci-test recipe should be detected (@ prefix stripped)")
	}
	if !graph.HasRecipe("ci-integration-test") {
		t.Error("ci-integration-test recipe should be detected (@ prefix stripped)")
	}

	// Verify regular recipe without @ prefix still works
	if !graph.HasRecipe("regular-recipe") {
		t.Error("regular-recipe should be detected")
	}
}

func TestParseJustfile_MixedFormat(t *testing.T) {
	content := `# Mixed format justfile

# Recipe with @ and no dependencies
@clean:
    rm -rf build

# Recipe with @ and dependencies
@test: build lint
    go test ./...

# Recipe without @ and with dependencies
build: clean
    go build

# Recipe without @ and no dependencies  
lint:
    golangci-lint run
`

	tmpfile, err := os.CreateTemp("", "justfile-mixed-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	graph, err := ParseJustfile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseJustfile failed: %v", err)
	}

	// Verify all recipes are detected with correct names (no @ prefix)
	expectedRecipes := map[string][]string{
		"clean": {},
		"test":  {"build", "lint"},
		"build": {"clean"},
		"lint":  {},
	}

	for name, expectedDeps := range expectedRecipes {
		recipe, ok := graph.Recipes[name]
		if !ok {
			t.Errorf("Recipe '%s' not found", name)
			continue
		}

		if len(recipe.Dependencies) != len(expectedDeps) {
			t.Errorf("Recipe '%s': expected %d dependencies, got %d", name, len(expectedDeps), len(recipe.Dependencies))
			continue
		}

		for i, dep := range expectedDeps {
			if i >= len(recipe.Dependencies) || recipe.Dependencies[i] != dep {
				t.Errorf("Recipe '%s': expected dependency '%s' at index %d, got '%v'", name, dep, i, recipe.Dependencies)
			}
		}
	}
}
