package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMakefile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantTargets map[string][]string // target -> dependencies
	}{
		{
			name: "simple targets",
			content: `build:
	go build

test:
	go test ./...
`,
			wantTargets: map[string][]string{
				"build": {},
				"test":  {},
			},
		},
		{
			name: "targets with dependencies",
			content: `build: deps
	go build

deps:
	go mod download

test: build
	go test ./...
`,
			wantTargets: map[string][]string{
				"build": {"deps"},
				"deps":  {},
				"test":  {"build"},
			},
		},
		{
			name: "multiple dependencies",
			content: `build: frontend backend
	@echo "done"

frontend:
	npm run build

backend:
	go build
`,
			wantTargets: map[string][]string{
				"build":    {"frontend", "backend"},
				"frontend": {},
				"backend":  {},
			},
		},
		{
			name: "ci- prefixed targets",
			content: `ci-build: ci-deps
	go build -ldflags="-s -w"

ci-deps:
	go mod download
	go mod verify

build:
	go build
`,
			wantTargets: map[string][]string{
				"ci-build": {"ci-deps"},
				"ci-deps":  {},
				"build":    {},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary Makefile
			tmpDir := t.TempDir()
			makefilePath := filepath.Join(tmpDir, "Makefile")
			if err := os.WriteFile(makefilePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp Makefile: %v", err)
			}

			// Parse Makefile
			graph, err := ParseMakefile(makefilePath)
			if err != nil {
				t.Fatalf("ParseMakefile() error = %v", err)
			}

			// Check targets
			if len(graph.Targets) != len(tt.wantTargets) {
				t.Errorf("got %d targets, want %d", len(graph.Targets), len(tt.wantTargets))
			}

			for targetName, wantDeps := range tt.wantTargets {
				target, exists := graph.Targets[targetName]
				if !exists {
					t.Errorf("target %q not found", targetName)
					continue
				}

				if len(target.Dependencies) != len(wantDeps) {
					t.Errorf("target %q: got %d dependencies, want %d",
						targetName, len(target.Dependencies), len(wantDeps))
					continue
				}

				for i, gotDep := range target.Dependencies {
					if i >= len(wantDeps) || gotDep != wantDeps[i] {
						t.Errorf("target %q: dependency[%d] = %q, want %q",
							targetName, i, gotDep, wantDeps[i])
					}
				}
			}
		})
	}
}

func TestGetDependencyTree(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		target   string
		wantTree []string
	}{
		{
			name: "single target no deps",
			content: `build:
	go build
`,
			target:   "build",
			wantTree: []string{"build"},
		},
		{
			name: "target with single dependency",
			content: `build: deps
	go build

deps:
	go mod download
`,
			target:   "build",
			wantTree: []string{"deps", "build"},
		},
		{
			name: "complex dependency chain",
			content: `build: compile
	@echo "done"

compile: generate
	go build

generate: deps
	go generate ./...

deps:
	go mod download
`,
			target:   "build",
			wantTree: []string{"deps", "generate", "compile", "build"},
		},
		{
			name: "parallel dependencies",
			content: `build: frontend backend
	@echo "done"

frontend: deps
	npm run build

backend: deps
	go build

deps:
	npm ci && go mod download
`,
			target:   "build",
			wantTree: []string{"deps", "frontend", "backend", "build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			makefilePath := filepath.Join(tmpDir, "Makefile")
			if err := os.WriteFile(makefilePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp Makefile: %v", err)
			}

			graph, err := ParseMakefile(makefilePath)
			if err != nil {
				t.Fatalf("ParseMakefile() error = %v", err)
			}

			gotTree := graph.GetDependencyTree(tt.target)

			if len(gotTree) != len(tt.wantTree) {
				t.Errorf("got tree length %d, want %d\ngot:  %v\nwant: %v",
					len(gotTree), len(tt.wantTree), gotTree, tt.wantTree)
				return
			}

			for i := range gotTree {
				if gotTree[i] != tt.wantTree[i] {
					t.Errorf("tree[%d] = %q, want %q", i, gotTree[i], tt.wantTree[i])
				}
			}
		})
	}
}

func TestSelectBuildTarget(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		requested  string
		wantTarget string
		wantErr    bool
	}{
		{
			name: "ci-build exists",
			content: `ci-build:
	go build -ldflags="-s -w"

build:
	go build
`,
			requested:  "build",
			wantTarget: "ci-build",
			wantErr:    false,
		},
		{
			name: "only build exists",
			content: `build:
	go build
`,
			requested:  "build",
			wantTarget: "build",
			wantErr:    false,
		},
		{
			name: "neither exists",
			content: `test:
	go test ./...
`,
			requested:  "build",
			wantTarget: "",
			wantErr:    true,
		},
		{
			name: "ci-test exists",
			content: `ci-test:
	go test -race ./...

test:
	go test ./...
`,
			requested:  "test",
			wantTarget: "ci-test",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			makefilePath := filepath.Join(tmpDir, "Makefile")
			if err := os.WriteFile(makefilePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to create temp Makefile: %v", err)
			}

			gotTarget, err := selectBuildTarget(makefilePath, tt.requested)

			if (err != nil) != tt.wantErr {
				t.Errorf("selectBuildTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && gotTarget != tt.wantTarget {
				t.Errorf("selectBuildTarget() = %q, want %q", gotTarget, tt.wantTarget)
			}
		})
	}
}

func TestGetTargetEmoji(t *testing.T) {
	tests := []struct {
		target string
		want   string
	}{
		{"build", "🔨"},
		{"ci-build", "🔨"},
		{"test", "🧪"},
		{"ci-test", "🧪"},
		{"clean", "🧹"},
		{"publish", "📦"},
		{"deploy", "📦"},
		{"deps", "📥"},
		{"lint", "✨"},
		{"unknown", "📋"},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := getTargetEmoji(tt.target)
			if got != tt.want {
				t.Errorf("getTargetEmoji(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestMakeSafeMermaidName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"build", "build"},
		{"ci-build", "ci_build"},
		{"build.test", "build_test"},
		{"build/docker", "build_docker"},
		{"build:test", "build_test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeSafeMermaidName(tt.name)
			if got != tt.want {
				t.Errorf("makeSafeMermaidName(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
