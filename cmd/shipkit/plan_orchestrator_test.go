package main

import (
	"encoding/json"
	"os"
	"testing"
)

// TestPlanOrchestrator tests build orchestrator detection in plan output
func TestPlanOrchestrator(t *testing.T) {
	tests := []planTestCase{
		{
			name:      "makefile",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				content := generateMockMakefile("ci-build", "ci-test", "lint", "ci-install")
				if err := os.WriteFile("Makefile", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create Makefile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "make",
				"has_makefile":       "true",
				"has_justfile":       "false",
				"has_taskfile":       "false",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				if _, err := os.Stat("Makefile"); os.IsNotExist(err) {
					t.Error("Makefile should exist but was not found")
				}
				// Read and parse plan.json to verify dependencies
				planData, err := os.ReadFile(getPlanPath())
				if err != nil {
					t.Fatalf("failed to read plan.json: %v", err)
				}

				var plan Plan
				if err := json.Unmarshal(planData, &plan); err != nil {
					t.Fatalf("failed to parse plan.json: %v", err)
				}

				// Verify all targets are present with exact names (no mangling)
				if plan.BuildTargets == nil {
					t.Fatal("BuildTargets should not be nil")
				}

				expectedTargets := []string{"help", "ci-build", "ci-test", "lint", "ci-install"}
				for _, target := range expectedTargets {
					if _, ok := plan.BuildTargets[target]; !ok {
						t.Errorf("Expected target '%s' not found in BuildTargets", target)
					}
				}

				// Verify ci-install has 3 dependencies with exact names
				if deps, ok := plan.BuildTargets["ci-install"]; ok {
					if len(deps) != 3 {
						t.Errorf("Expected ci-install to have 3 dependencies, got %d: %v", len(deps), deps)
					}
					// Verify exact names: ci-build, ci-test, lint (NOT ci-lint - proving no prefix mangling)
					expected := map[string]bool{"ci-build": false, "ci-test": false, "lint": false}
					for _, dep := range deps {
						if _, ok := expected[dep]; ok {
							expected[dep] = true
						}
					}
					for name, found := range expected {
						if !found {
							t.Errorf("Expected dependency '%s' not found in ci-install deps: %v", name, deps)
						}
					}
				} else {
					t.Error("Expected ci-install to be in BuildTargets")
				}

				// Verify targets without deps have empty arrays
				for _, target := range []string{"help", "ci-build", "ci-test", "lint"} {
					if deps, ok := plan.BuildTargets[target]; ok {
						if len(deps) != 0 {
							t.Errorf("Expected target '%s' to have no dependencies, got %v", target, deps)
						}
					}
				}
			},
		},
		{
			name:      "justfile",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				content := generateMockJustfile("ci-build", "ci-test", "ci-docker")
				if err := os.WriteFile("justfile", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create justfile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "just",
				"has_makefile":       "false",
				"has_justfile":       "true",
				"has_taskfile":       "false",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				if _, err := os.Stat("justfile"); os.IsNotExist(err) {
					t.Error("justfile should exist but was not found")
				}
				// Verify recipes are in the file
				content, _ := os.ReadFile("justfile")
				contentStr := string(content)
				for _, recipe := range []string{"ci-build", "ci-test", "ci-docker"} {
					if !containsRecipe(contentStr, recipe) {
						t.Errorf("Expected justfile to have '%s' recipe", recipe)
					}
				}
			},
		},
		{
			name:      "taskfile",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				content := generateMockTaskfile("ci-build", "ci-test", "ci-clean")
				if err := os.WriteFile("Taskfile.yml", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create Taskfile.yml: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "task",
				"has_makefile":       "false",
				"has_justfile":       "false",
				"has_taskfile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				if _, err := os.Stat("Taskfile.yml"); os.IsNotExist(err) {
					t.Error("Taskfile.yml should exist but was not found")
				}
				// Verify tasks are in the file
				content, _ := os.ReadFile("Taskfile.yml")
				contentStr := string(content)
				for _, task := range []string{"ci-build", "ci-test", "ci-clean"} {
					if !containsTask(contentStr, task) {
						t.Errorf("Expected Taskfile.yml to have '%s' task", task)
					}
				}
			},
		},
		{
			name:      "convention_no_orchestrator",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			expectation: map[string]string{
				"build_orchestrator": "convention",
				"has_makefile":       "false",
				"has_justfile":       "false",
				"has_taskfile":       "false",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				// Verify no orchestrator files exist
				if _, err := os.Stat("Makefile"); !os.IsNotExist(err) {
					t.Error("Makefile should not exist in convention mode")
				}
				if _, err := os.Stat("justfile"); !os.IsNotExist(err) {
					t.Error("justfile should not exist in convention mode")
				}
				if _, err := os.Stat("Taskfile.yml"); !os.IsNotExist(err) {
					t.Error("Taskfile.yml should not exist in convention mode")
				}
			},
		},
		{
			name:      "makefile_and_justfile_priority",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				makefileContent := generateMockMakefile("build", "ci-build", "test")
				if err := os.WriteFile("Makefile", []byte(makefileContent), 0644); err != nil {
					t.Fatalf("failed to create Makefile: %v", err)
				}
				justfileContent := generateMockJustfile("build", "test")
				if err := os.WriteFile("justfile", []byte(justfileContent), 0644); err != nil {
					t.Fatalf("failed to create justfile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "make", // make has priority over just
				"has_makefile":       "true",
				"has_justfile":       "true",
				"has_taskfile":       "false",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				// Both files should exist
				if _, err := os.Stat("Makefile"); os.IsNotExist(err) {
					t.Error("Makefile should exist")
				}
				if _, err := os.Stat("justfile"); os.IsNotExist(err) {
					t.Error("justfile should exist")
				}
				// But make should be chosen due to priority
				if outputs["build_orchestrator"] != "make" {
					t.Errorf("Expected make to have priority, got: %s", outputs["build_orchestrator"])
				}
				// Verify Makefile has ci-build target
				if graph, err := ParseMakefile("Makefile"); err == nil {
					if !graph.HasTarget("ci-build") {
						t.Error("Expected Makefile to have 'ci-build' target")
					}
				}
			},
		},
		{
			name:      "justfile_and_taskfile_priority",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				if err := os.WriteFile("justfile", []byte(generateMockJustfile("build", "test")), 0644); err != nil {
					t.Fatalf("failed to create justfile: %v", err)
				}
				if err := os.WriteFile("Taskfile.yml", []byte(generateMockTaskfile("build", "test", "docker")), 0644); err != nil {
					t.Fatalf("failed to create Taskfile.yml: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "just", // just has priority over task
				"has_makefile":       "false",
				"has_justfile":       "true",
				"has_taskfile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				// Both files should exist
				if _, err := os.Stat("justfile"); os.IsNotExist(err) {
					t.Error("justfile should exist")
				}
				if _, err := os.Stat("Taskfile.yml"); os.IsNotExist(err) {
					t.Error("Taskfile.yml should exist")
				}
				// But just should be chosen due to priority
				if outputs["build_orchestrator"] != "just" {
					t.Errorf("Expected just to have priority, got: %s", outputs["build_orchestrator"])
				}
			},
		},
		{
			name:      "all_orchestrators_priority",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				makefileContent := generateMockMakefile("build", "ci-build", "test", "ci-test", "docker")
				if err := os.WriteFile("Makefile", []byte(makefileContent), 0644); err != nil {
					t.Fatalf("failed to create Makefile: %v", err)
				}
				if err := os.WriteFile("justfile", []byte(generateMockJustfile("build", "test")), 0644); err != nil {
					t.Fatalf("failed to create justfile: %v", err)
				}
				if err := os.WriteFile("Taskfile.yml", []byte(generateMockTaskfile("build", "test", "lint")), 0644); err != nil {
					t.Fatalf("failed to create Taskfile.yml: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "make", // make has highest priority
				"has_makefile":       "true",
				"has_justfile":       "true",
				"has_taskfile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				// All files should exist
				for _, file := range []string{"Makefile", "justfile", "Taskfile.yml"} {
					if _, err := os.Stat(file); os.IsNotExist(err) {
						t.Errorf("%s should exist", file)
					}
				}
				// But make should be chosen due to highest priority
				if outputs["build_orchestrator"] != "make" {
					t.Errorf("Expected make to have highest priority when all exist, got: %s", outputs["build_orchestrator"])
				}
				// Verify Makefile has all expected targets
				if graph, err := ParseMakefile("Makefile"); err == nil {
					for _, target := range []string{"build", "ci-build", "test", "ci-test", "docker"} {
						if !graph.HasTarget(target) {
							t.Errorf("Expected Makefile to have '%s' target", target)
						}
					}
				}
			},
		},
	}

	runPlanTests(t, tests)
}

// TestPlanMakefileTargets tests that the plan can parse Makefile targets
func TestPlanMakefileTargets(t *testing.T) {
	tests := []struct {
		name            string
		makefileContent string
		expectedTargets []string
	}{
		{
			name:            "build_and_test_targets",
			makefileContent: generateMockMakefile("build", "test"),
			expectedTargets: []string{"help", "build", "test"},
		},
		{
			name:            "ci_targets",
			makefileContent: generateMockMakefile("build", "ci-build", "test", "ci-test"),
			expectedTargets: []string{"help", "build", "ci-build", "test", "ci-test"},
		},
		{
			name:            "docker_target",
			makefileContent: generateMockMakefile("build", "docker", "clean"),
			expectedTargets: []string{"help", "build", "docker", "clean"},
		},
		{
			name:            "full_targets",
			makefileContent: generateMockMakefile("build", "ci-build", "test", "ci-test", "docker", "clean", "install", "lint"),
			expectedTargets: []string{"help", "build", "ci-build", "test", "ci-test", "docker", "clean", "install", "lint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for this test
			tempDir := t.TempDir()
			makefilePath := tempDir + "/Makefile"

			// Write Makefile
			if err := os.WriteFile(makefilePath, []byte(tt.makefileContent), 0644); err != nil {
				t.Fatalf("failed to create Makefile: %v", err)
			}

			// Parse the Makefile
			graph, err := ParseMakefile(makefilePath)
			if err != nil {
				t.Fatalf("failed to parse Makefile: %v", err)
			}

			// Verify all expected targets are present
			targets := graph.GetTargets()
			targetMap := make(map[string]bool)
			for _, target := range targets {
				targetMap[target] = true
			}

			for _, expectedTarget := range tt.expectedTargets {
				if !targetMap[expectedTarget] {
					t.Errorf("Expected target '%s' not found in parsed Makefile. Found targets: %v", expectedTarget, targets)
				}
			}

			// Verify HasTarget works correctly
			for _, expectedTarget := range tt.expectedTargets {
				if !graph.HasTarget(expectedTarget) {
					t.Errorf("HasTarget('%s') returned false, expected true", expectedTarget)
				}
			}
		})
	}
}

// TestPlanJustfileRecipes tests that justfile recipes are properly generated
func TestPlanJustfileRecipes(t *testing.T) {
	tests := []struct {
		name            string
		justfileContent string
		expectedRecipes []string
	}{
		{
			name:            "build_and_test_recipes",
			justfileContent: generateMockJustfile("build", "test"),
			expectedRecipes: []string{"help", "build", "test"},
		},
		{
			name:            "ci_recipes",
			justfileContent: generateMockJustfile("build", "ci-build", "test", "ci-test"),
			expectedRecipes: []string{"help", "build", "ci-build", "test", "ci-test"},
		},
		{
			name:            "docker_recipe",
			justfileContent: generateMockJustfile("build", "docker", "clean"),
			expectedRecipes: []string{"help", "build", "docker", "clean"},
		},
		{
			name:            "full_recipes",
			justfileContent: generateMockJustfile("build", "ci-build", "test", "ci-test", "docker", "clean", "install", "lint"),
			expectedRecipes: []string{"help", "build", "ci-build", "test", "ci-test", "docker", "clean", "install", "lint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for this test
			tempDir := t.TempDir()
			justfilePath := tempDir + "/justfile"

			// Write justfile
			if err := os.WriteFile(justfilePath, []byte(tt.justfileContent), 0644); err != nil {
				t.Fatalf("failed to create justfile: %v", err)
			}

			// Verify file was created with expected content
			content, err := os.ReadFile(justfilePath)
			if err != nil {
				t.Fatalf("failed to read justfile: %v", err)
			}

			// Verify recipes appear in the file content
			contentStr := string(content)
			for _, recipe := range tt.expectedRecipes {
				if recipe == "help" || recipe == "default" {
					continue // Skip checking these as they're in the header
				}
				// Check that recipe name appears followed by a colon
				if !containsRecipe(contentStr, recipe) {
					t.Errorf("Expected recipe '%s' not found in justfile content", recipe)
				}
			}
		})
	}
}

// TestPlanTaskfileTasks tests that Taskfile tasks are properly generated
func TestPlanTaskfileTasks(t *testing.T) {
	tests := []struct {
		name            string
		taskfileContent string
		expectedTasks   []string
	}{
		{
			name:            "build_and_test_tasks",
			taskfileContent: generateMockTaskfile("build", "test"),
			expectedTasks:   []string{"default", "build", "test"},
		},
		{
			name:            "ci_tasks",
			taskfileContent: generateMockTaskfile("build", "ci-build", "test", "ci-test"),
			expectedTasks:   []string{"default", "build", "ci-build", "test", "ci-test"},
		},
		{
			name:            "docker_task",
			taskfileContent: generateMockTaskfile("build", "docker", "clean"),
			expectedTasks:   []string{"default", "build", "docker", "clean"},
		},
		{
			name:            "full_tasks",
			taskfileContent: generateMockTaskfile("build", "ci-build", "test", "ci-test", "docker", "clean", "install", "lint"),
			expectedTasks:   []string{"default", "build", "ci-build", "test", "ci-test", "docker", "clean", "install", "lint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for this test
			tempDir := t.TempDir()
			taskfilePath := tempDir + "/Taskfile.yml"

			// Write Taskfile
			if err := os.WriteFile(taskfilePath, []byte(tt.taskfileContent), 0644); err != nil {
				t.Fatalf("failed to create Taskfile.yml: %v", err)
			}

			// Verify file was created with expected content
			content, err := os.ReadFile(taskfilePath)
			if err != nil {
				t.Fatalf("failed to read Taskfile.yml: %v", err)
			}

			// Verify tasks appear in the file content
			contentStr := string(content)
			for _, task := range tt.expectedTasks {
				if !containsTask(contentStr, task) {
					t.Errorf("Expected task '%s' not found in Taskfile.yml content", task)
				}
			}
		})
	}
}

// generateMockMakefile creates a realistic Makefile with common targets
func generateMockMakefile(targets ...string) string {
	content := `# Auto-generated Makefile
.PHONY: help clean build test docker ci-build ci-test

help: ## Show this help
@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

`

	// Add requested targets
	for _, target := range targets {
		switch target {
		case "build":
			content += `build: ## Build the project
@echo "Building..."
go build -o bin/app ./cmd/app

`
		case "ci-build":
			content += `ci-build: ## Build the project (CI mode)
@echo "CI Building..."
go build -v -o bin/app ./cmd/app

`
		case "test":
			content += `test: ## Run tests
@echo "Running tests..."
go test -v ./...

`
		case "ci-test":
			content += `ci-test: ## Run tests (CI mode)
@echo "CI Testing..."
go test -v -race -coverprofile=coverage.out ./...

`
		case "docker":
			content += `docker: ## Build docker image
@echo "Building docker image..."
docker build -t myapp:latest .

`
		case "clean":
			content += `clean: ## Clean build artifacts
@echo "Cleaning..."
rm -rf bin/ dist/

`
		case "install":
			content += `install: build ## Install the binary
@echo "Installing..."
cp bin/app /usr/local/bin/

`
		case "lint":
			content += `lint: ## Run linter
@echo "Linting..."
golangci-lint run

`
		case "ci-lint":
			content += `ci-lint: ## Run linter (CI mode)
@echo "CI Linting..."
golangci-lint run --timeout=5m

`
		case "ci-install":
			content += `ci-install: ci-build ci-test lint ## CI: Install after verifying build, tests, and linting
@echo "CI Installing..."
cp bin/app /usr/local/bin/

`
		}
	}

	return content
}

// generateMockJustfile creates a realistic justfile with common recipes
func generateMockJustfile(recipes ...string) string {
	content := `# Auto-generated justfile
default: help

# Show this help
help:
    @just --list

`

	// Add requested recipes
	for _, recipe := range recipes {
		switch recipe {
		case "build":
			content += `# Build the project
build:
    @echo "Building..."
    go build -o bin/app ./cmd/app

`
		case "ci-build":
			content += `# Build the project (CI mode)
ci-build:
    @echo "CI Building..."
    go build -v -o bin/app ./cmd/app

`
		case "test":
			content += `# Run tests
test:
    @echo "Running tests..."
    go test -v ./...

`
		case "ci-test":
			content += `# Run tests (CI mode)
ci-test:
    @echo "CI Testing..."
    go test -v -race -coverprofile=coverage.out ./...

`
		case "docker":
			content += `# Build docker image
docker:
    @echo "Building docker image..."
    docker build -t myapp:latest .

`
		case "ci-docker":
			content += `# Build docker image (CI mode)
ci-docker:
    @echo "CI Building docker image..."
    docker build -t myapp:latest .

`
		case "clean":
			content += `# Clean build artifacts
clean:
    @echo "Cleaning..."
    rm -rf bin/ dist/

`
		case "ci-clean":
			content += `# Clean build artifacts (CI mode)
ci-clean:
    @echo "CI Cleaning..."
    rm -rf bin/ dist/

`
		case "install":
			content += `# Install the binary
install: build
    @echo "Installing..."
    cp bin/app /usr/local/bin/

`
		case "lint":
			content += `# Run linter
lint:
    @echo "Linting..."
    golangci-lint run

`
		case "ci-lint":
			content += `# Run linter (CI mode)
ci-lint:
    @echo "CI Linting..."
    golangci-lint run --timeout=5m

`
		case "ci-install":
			content += `# CI: Install after verifying build, tests, and linting
ci-install: ci-build ci-test ci-lint
    @echo "CI Installing..."
    cp bin/app /usr/local/bin/

`
		}
	}

	return content
}

// generateMockTaskfile creates a realistic Taskfile.yml with common tasks
func generateMockTaskfile(tasks ...string) string {
	content := `# Auto-generated Taskfile
version: '3'

tasks:
  default:
    desc: List available tasks
    cmds:
      - task --list

`

	// Add requested tasks
	for _, task := range tasks {
		switch task {
		case "build":
			content += `  build:
    desc: Build the project
    cmds:
      - echo "Building..."
      - go build -o bin/app ./cmd/app

`
		case "ci-build":
			content += `  ci-build:
    desc: Build the project (CI mode)
    cmds:
      - echo "CI Building..."
      - go build -v -o bin/app ./cmd/app

`
		case "test":
			content += `  test:
    desc: Run tests
    cmds:
      - echo "Running tests..."
      - go test -v ./...

`
		case "ci-test":
			content += `  ci-test:
    desc: Run tests (CI mode)
    cmds:
      - echo "CI Testing..."
      - go test -v -race -coverprofile=coverage.out ./...

`
		case "docker":
			content += `  docker:
    desc: Build docker image
    cmds:
      - echo "Building docker image..."
      - docker build -t myapp:latest .

`
		case "ci-docker":
			content += `  ci-docker:
    desc: Build docker image (CI mode)
    cmds:
      - echo "CI Building docker image..."
      - docker build -t myapp:latest .

`
		case "clean":
			content += `  clean:
    desc: Clean build artifacts
    cmds:
      - echo "Cleaning..."
      - rm -rf bin/ dist/

`
		case "ci-clean":
			content += `  ci-clean:
    desc: Clean build artifacts (CI mode)
    cmds:
      - echo "CI Cleaning..."
      - rm -rf bin/ dist/

`
		case "install":
			content += `  install:
    desc: Install the binary
    deps: [build]
    cmds:
      - echo "Installing..."
      - cp bin/app /usr/local/bin/

`
		case "lint":
			content += `  lint:
    desc: Run linter
    cmds:
      - echo "Linting..."
      - golangci-lint run

`
		case "ci-lint":
			content += `  ci-lint:
    desc: Run linter (CI mode)
    cmds:
      - echo "CI Linting..."
      - golangci-lint run --timeout=5m

`
		case "ci-install":
			content += `  ci-install:
    desc: CI - Install after verifying build, tests, and linting
    deps: [ci-build, ci-test, ci-lint]
    cmds:
      - echo "CI Installing..."
      - cp bin/app /usr/local/bin/

`
		}
	}

	return content
}

// containsTask checks if a task is defined in Taskfile content
func containsTask(content, task string) bool {
	// Check for task definition in YAML format: "  taskname:" at start of line
	lines := splitLines(content)
	for _, line := range lines {
		trimmed := trimSpace(line)
		// In Taskfile, tasks are indented under "tasks:" section
		if trimmed == task+":" || startsWith(trimmed, task+":") {
			return true
		}
	}
	return false
}

// containsRecipe checks if a recipe is defined in justfile content
func containsRecipe(content, recipe string) bool {
	// Simple check: recipe name followed by colon at start of line or after comment
	lines := splitLines(content)
	for _, line := range lines {
		trimmed := trimSpace(line)
		if trimmed == recipe+":" || startsWith(trimmed, recipe+":") {
			return true
		}
	}
	return false
}

// Helper functions to avoid importing strings package
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for start < end && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// TestPlanBuildTargetsExtraction tests that CI targets and dependencies are extracted
func TestPlanBuildTargetsExtraction(t *testing.T) {
	tests := []planTestCase{
		{
			name:      "makefile_with_ci_targets",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				content := generateMockMakefile("build", "ci-build", "test", "ci-test", "docker", "clean")
				if err := os.WriteFile("Makefile", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create Makefile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "make",
				"has_makefile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				// Verify Makefile exists
				if _, err := os.Stat("Makefile"); os.IsNotExist(err) {
					t.Error("Makefile should exist but was not found")
				}

				// Read and parse plan.json to check BuildTargets
				planData, err := os.ReadFile(getPlanPath())
				if err != nil {
					t.Fatalf("failed to read plan.json: %v", err)
				}

				var plan Plan
				if err := json.Unmarshal(planData, &plan); err != nil {
					t.Fatalf("failed to parse plan.json: %v", err)
				}

				// Verify CI targets are present
				expectedTargets := []string{"ci-build", "ci-test", "build", "test", "docker", "clean"}
				if plan.BuildTargets == nil {
					t.Fatal("BuildTargets should not be nil")
				}

				for _, target := range expectedTargets {
					if _, ok := plan.BuildTargets[target]; !ok {
						t.Errorf("Expected target '%s' not found in BuildTargets", target)
					}
				}

				// Log all targets
				t.Logf("BuildTargets: %v", plan.BuildTargets)
			},
		},
		{
			name:      "makefile_with_target_dependencies",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				content := generateMockMakefile("build", "ci-build", "test", "install")
				if err := os.WriteFile("Makefile", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create Makefile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "make",
				"has_makefile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				// Read and parse plan.json
				planData, err := os.ReadFile(getPlanPath())
				if err != nil {
					t.Fatalf("failed to read plan.json: %v", err)
				}

				var plan Plan
				if err := json.Unmarshal(planData, &plan); err != nil {
					t.Fatalf("failed to parse plan.json: %v", err)
				}

				// Verify install target depends on build
				if plan.BuildTargets == nil {
					t.Error("BuildTargets should not be nil")
					return
				}

				if deps, ok := plan.BuildTargets["install"]; ok {
					if len(deps) != 1 || deps[0] != "build" {
						t.Errorf("Expected install to depend on ['build'], got %v", deps)
					}
				} else {
					t.Error("Expected install target to have dependencies")
				}
			},
		},
		{
			name:      "makefile_with_multiple_dependencies",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				// Create a Makefile with a target that has multiple dependencies
				content := `# Makefile with multiple dependencies
.PHONY: build test clean install ci-all

build: ## Build the project
	@echo "Building..."
	go build -o bin/app ./cmd/app

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/ dist/

install: build test ## Install after building and testing
	@echo "Installing..."
	cp bin/app /usr/local/bin/

ci-build: build ## CI build depends on build
	@echo "CI Building..."
	go build -v -o bin/app ./cmd/app
`
				if err := os.WriteFile("Makefile", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create Makefile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "make",
				"has_makefile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				// Read and parse plan.json
				planData, err := os.ReadFile(getPlanPath())
				if err != nil {
					t.Fatalf("failed to read plan.json: %v", err)
				}

				var plan Plan
				if err := json.Unmarshal(planData, &plan); err != nil {
					t.Fatalf("failed to parse plan.json: %v", err)
				}

				// Verify install target has multiple dependencies
				if plan.BuildTargets == nil {
					t.Error("BuildTargets should not be nil")
					return
				}

				if deps, ok := plan.BuildTargets["install"]; ok {
					if len(deps) != 2 {
						t.Errorf("Expected install to have 2 dependencies, got %d: %v", len(deps), deps)
					}
					// Check that both build and test are in the dependencies
					hassBuild := false
					hasTest := false
					for _, dep := range deps {
						if dep == "build" {
							hassBuild = true
						}
						if dep == "test" {
							hasTest = true
						}
					}
					if !hassBuild || !hasTest {
						t.Errorf("Expected install to depend on both 'build' and 'test', got %v", deps)
					}
				} else {
					t.Error("Expected install target to have dependencies")
				}

				// Verify ci-build depends on build
				if deps, ok := plan.BuildTargets["ci-build"]; ok {
					if len(deps) != 1 || deps[0] != "build" {
						t.Errorf("Expected ci-build to depend on ['build'], got %v", deps)
					}
				}
			},
		},
		{
			name:      "justfile_with_recipes",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				content := generateMockJustfile("ci-build", "ci-test", "ci-install")
				if err := os.WriteFile("justfile", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create justfile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "just",
				"has_justfile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				planData, err := os.ReadFile(getPlanPath())
				if err != nil {
					t.Fatalf("failed to read plan.json: %v", err)
				}

				var plan Plan
				if err := json.Unmarshal(planData, &plan); err != nil {
					t.Fatalf("failed to parse plan.json: %v", err)
				}

				// Verify ci-install has multiple dependencies
				if plan.BuildTargets != nil {
					if deps, ok := plan.BuildTargets["ci-install"]; ok {
						t.Logf("ci-install dependencies: %v", deps)
						if len(deps) < 2 {
							t.Errorf("Expected ci-install to have multiple dependencies, got %d: %v", len(deps), deps)
						}
					}
				}
			},
		},
		{
			name:      "taskfile_with_tasks",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				content := generateMockTaskfile("ci-build", "ci-test", "ci-install")
				if err := os.WriteFile("Taskfile.yml", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create Taskfile.yml: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "task",
				"has_taskfile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				// Read and parse plan.json
				planData, err := os.ReadFile(getPlanPath())
				if err != nil {
					t.Fatalf("failed to read plan.json: %v", err)
				}

				var plan Plan
				if err := json.Unmarshal(planData, &plan); err != nil {
					t.Fatalf("failed to parse plan.json: %v", err)
				}

				// Verify Taskfile tasks are extracted
				if plan.BuildTargets == nil {
					t.Fatal("BuildTargets should not be nil")
				}

				// Verify ci-install has multiple dependencies
				if plan.BuildTargets != nil {
					if deps, ok := plan.BuildTargets["ci-install"]; ok {
						t.Logf("ci-install dependencies: %v", deps)
						if len(deps) < 2 {
							t.Errorf("Expected ci-install to have multiple dependencies, got %d: %v", len(deps), deps)
						}
					}
				}
			},
		},
		{
			name:      "makefile_with_ci_all_multiple_dependencies",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				// Create a Makefile with ci-all that depends on multiple targets
				content := `# Makefile with multiple dependencies
.PHONY: build test lint clean ci-all

build: ## Build the project
	@echo "Building..."
	go build -o bin/app ./cmd/app

test: ## Run tests
	@echo "Running tests..."
	go test -v ./...

lint: ## Run linter
	@echo "Linting..."
	golangci-lint run

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/ dist/

ci-all: build test lint ## Run all CI checks (build, test, lint)
	@echo "All CI checks passed!"
`
				if err := os.WriteFile("Makefile", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create Makefile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "make",
				"has_makefile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				// Read and parse plan.json
				planData, err := os.ReadFile(getPlanPath())
				if err != nil {
					t.Fatalf("failed to read plan.json: %v", err)
				}

				var plan Plan
				if err := json.Unmarshal(planData, &plan); err != nil {
					t.Fatalf("failed to parse plan.json: %v", err)
				}

				// Verify ci-all target has multiple dependencies
				if plan.BuildTargets == nil {
					t.Error("BuildTargets should not be nil")
					return
				}

				if deps, ok := plan.BuildTargets["ci-all"]; ok {
					if len(deps) != 3 {
						t.Errorf("Expected ci-all to have 3 dependencies, got %d: %v", len(deps), deps)
					}
					// Check that build, test, and lint are all in the dependencies
					hasBuild := false
					hasTest := false
					hasLint := false
					for _, dep := range deps {
						switch dep {
						case "build":
							hasBuild = true
						case "test":
							hasTest = true
						case "lint":
							hasLint = true
						}
					}
					if !hasBuild || !hasTest || !hasLint {
						t.Errorf("Expected ci-all to depend on ['build', 'test', 'lint'], got %v", deps)
					}
				} else {
					t.Error("Expected ci-all target to have dependencies")
				}
			},
		},
		{
			name:      "makefile_with_descriptive_target_names",
			eventName: "workflow_dispatch",
			plan: &Plan{
				Mode:                "release",
				Bump:                "patch",
				DryRun:              true,
				UseDocker:           true,
				UseGoreleaser:       true,
				UseGoreleaserDocker: true,
				DockerImage:         "karloie/kompass",
			},
			mockLatest: "v1.2.2",
			setupFunc: func(t *testing.T) {
				// Create Makefile with descriptively-named dependencies
				content := `# Makefile with descriptive target names
.PHONY: compile-binary run-test-suite check-code-quality clean ci-full-pipeline

compile-binary: ## Compile the application
	@echo "Compiling..."
	go build -o bin/app ./cmd/app

run-test-suite: ## Run all tests
	@echo "Testing..."
	go test -v ./...

check-code-quality: ## Static analysis
	@echo "Linting..."
	golangci-lint run

clean: ## Clean artifacts
	@echo "Cleaning..."
	rm -rf bin/ dist/

ci-full-pipeline: compile-binary run-test-suite check-code-quality ## CI pipeline with all checks
	@echo "Pipeline complete!"
`
				if err := os.WriteFile("Makefile", []byte(content), 0644); err != nil {
					t.Fatalf("failed to create Makefile: %v", err)
				}
			},
			expectation: map[string]string{
				"build_orchestrator": "make",
				"has_makefile":       "true",
			},
			validateFunc: func(t *testing.T, outputs map[string]string) {
				planData, err := os.ReadFile(getPlanPath())
				if err != nil {
					t.Fatalf("failed to read plan.json: %v", err)
				}

				var plan Plan
				if err := json.Unmarshal(planData, &plan); err != nil {
					t.Fatalf("failed to parse plan.json: %v", err)
				}

				// Verify ci-full-pipeline has descriptively-named dependencies
				if plan.BuildTargets == nil {
					t.Error("BuildTargets should not be nil")
					return
				}

				if deps, ok := plan.BuildTargets["ci-full-pipeline"]; ok {
					expected := []string{"compile-binary", "run-test-suite", "check-code-quality"}
					if len(deps) != 3 {
						t.Errorf("Expected 3 dependencies, got %d: %v", len(deps), deps)
					}
					// Verify the descriptive names are preserved
					for _, expectedDep := range expected {
						found := false
						for _, actualDep := range deps {
							if actualDep == expectedDep {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("Expected dependency '%s' not found in %v", expectedDep, deps)
						}
					}
				} else {
					t.Error("Expected ci-full-pipeline to have dependencies")
				}
			},
		},
	}

	runPlanTests(t, tests)
}
