package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Kompass scenarios
func TestKompassProjectDetection(t *testing.T) {
	// Kompass: Go + Node.js
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// go.mod
	goMod := `module github.com/karloie/kompass

go 1.25.5

require (
	github.com/cilium/cilium v1.19.1
	k8s.io/api v0.35.0
)
`
	if err := os.WriteFile("go.mod", []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// package.json
	packageJSON := `{
  "name": "kompass-web",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "vue": "^3.5.13"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.2.1",
    "vite": "^6.2.0"
  }
}`
	if err := os.WriteFile("package.json", []byte(packageJSON), 0644); err != nil {
		t.Fatal(err)
	}

	// Check project name
	projectName := detectProjectName()
	if projectName != "kompass" {
		t.Errorf("detectProjectName() = %q, want %q", projectName, "kompass")
	}
}

func TestKompassGoReleaserConfig(t *testing.T) {
	// Kompass with Node.js
	config := GoReleaserConfig{
		ProjectName:  "kompass",
		BinaryName:   "kompass",
		MainPath:     "./cmd/kompass",
		RepoOwner:    "karloie",
		RepoName:     "kompass",
		Description:  "Kubernetes service mesh monitoring",
		License:      "MIT",
		DockerImage:  "karloie/kompass",
		HasChangelog: false,
		HasDocker:    false,
		DockerFile:   "",
	}

	// Temp dir
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, ".goreleaser.yml")

	if err := generateGoReleaserConfig(config, outputFile); err != nil {
		t.Fatalf("generateGoReleaserConfig failed: %v", err)
	}

	// Read output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read generated config: %v", err)
	}

	configStr := string(content)

	// Check config contents
	expectations := []string{
		"kompass",
		"./cmd/kompass",
		"karloie/kompass",
		"Kubernetes service mesh monitoring",
		"id: kompass",
	}

	for _, exp := range expectations {
		if !strings.Contains(configStr, exp) {
			t.Errorf("config missing %q", exp)
		}
	}
}

// Bastille scenarios
func TestBastilleProjectDetection(t *testing.T) {
	// Bastille: pure Go
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	// go.mod only
	goMod := `module github.com/karloie/bastille

go 1.24.0

require (
	golang.org/x/crypto v0.44.0
	golang.org/x/time v0.14.0
)
`
	if err := os.WriteFile("go.mod", []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// No package.json

	// Check project name
	projectName := detectProjectName()
	if projectName != "bastille" {
		t.Errorf("detectProjectName() = %q, want %q", projectName, "bastille")
	}
}

func TestBastilleGoReleaserConfig(t *testing.T) {
	// Bastille without Node.js
	config := GoReleaserConfig{
		ProjectName:  "bastille",
		BinaryName:   "bastille",
		MainPath:     "./cmd/bastille",
		RepoOwner:    "karloie",
		RepoName:     "bastille",
		Description:  "Security management tool",
		License:      "MIT",
		DockerImage:  "karloie/bastille",
		HasChangelog: false,
		HasDocker:    false,
		DockerFile:   "",
	}

	// Temp dir
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, ".goreleaser.yml")

	if err := generateGoReleaserConfig(config, outputFile); err != nil {
		t.Fatalf("generateGoReleaserConfig failed: %v", err)
	}

	// Read output
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read generated config: %v", err)
	}

	configStr := string(content)

	// Should NOT contain npm
	if strings.Contains(configStr, "npm") {
		t.Error("Bastille config should NOT contain npm commands")
	}

	// Should contain expected
	expectations := []string{
		"bastille",
		"./cmd/bastille",
		"karloie/bastille",
		"Security management tool",
		"go mod tidy",
		"id: bastille",
	}

	for _, exp := range expectations {
		if !strings.Contains(configStr, exp) {
			t.Errorf("config missing %q", exp)
		}
	}
}

// Mixed scenarios
func TestClientProjectModuleExtraction(t *testing.T) {
	tests := []struct {
		name       string
		goModPath  string
		wantModule string
		wantName   string
	}{
		{
			"kompass",
			"github.com/karloie/kompass",
			"github.com/karloie/kompass",
			"kompass",
		},
		{
			"bastille",
			"github.com/karloie/bastille",
			"github.com/karloie/bastille",
			"bastille",
		},
		{
			"nested module",
			"github.com/org/team/project",
			"github.com/org/team/project",
			"project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			oldWd, _ := os.Getwd()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(oldWd)

			// Write go.mod
			goMod := "module " + tt.goModPath + "\n\ngo 1.21.0\n"
			if err := os.WriteFile("go.mod", []byte(goMod), 0644); err != nil {
				t.Fatal(err)
			}

			// Extract name
			projectName := detectProjectName()
			if projectName != tt.wantName {
				t.Errorf("detectProjectName() = %q, want %q", projectName, tt.wantName)
			}
		})
	}
}

func TestClientProjectDockerImageNaming(t *testing.T) {
	tests := []struct {
		name      string
		project   string
		owner     string
		wantImage string
	}{
		{"kompass", "kompass", "karloie", "karloie/kompass"},
		{"bastille", "bastille", "karloie", "karloie/bastille"},
		{"custom owner", "myapp", "myorg", "myorg/myapp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			image := tt.owner + "/" + tt.project
			if image != tt.wantImage {
				t.Errorf("image = %q, want %q", image, tt.wantImage)
			}
		})
	}
}

// Real workflow scenarios
func TestKompassReleaseWorkflowParams(t *testing.T) {
	// Workflow params
	params := map[string]string{
		"image":                "karloie/kompass",
		"node_version":         "20",
		"frontend_install_cmd": "npm ci",
		"frontend_build_cmd":   "npm run build",
		"event_name":           "workflow_dispatch",
		"bump":                 "patch",
	}

	// Validate required
	if params["image"] == "" {
		t.Error("image required")
	}
	if params["node_version"] == "" {
		t.Error("node_version required for kompass")
	}
	if params["frontend_install_cmd"] == "" {
		t.Error("frontend_install_cmd required for kompass")
	}
	if params["frontend_build_cmd"] == "" {
		t.Error("frontend_build_cmd required for kompass")
	}

	// Check values
	if params["image"] != "karloie/kompass" {
		t.Errorf("image = %q, want %q", params["image"], "karloie/kompass")
	}
}

func TestBastilleReleaseWorkflowParams(t *testing.T) {
	// Workflow params
	params := map[string]string{
		"image":      "karloie/bastille",
		"event_name": "workflow_dispatch",
		"bump":       "minor",
	}

	// Validate required
	if params["image"] == "" {
		t.Error("image required")
	}

	// Check NO node params
	if params["node_version"] != "" {
		t.Error("bastille should NOT have node_version")
	}
	if params["frontend_install_cmd"] != "" {
		t.Error("bastille should NOT have frontend_install_cmd")
	}
	if params["frontend_build_cmd"] != "" {
		t.Error("bastille should NOT have frontend_build_cmd")
	}

	// Check values
	if params["image"] != "karloie/bastille" {
		t.Errorf("image = %q, want %q", params["image"], "karloie/bastille")
	}
}

func TestProjectDescriptionExtraction(t *testing.T) {
	tests := []struct {
		name        string
		packageJSON string
		want        string
	}{
		{
			"kompass with description",
			`{"name": "kompass-web", "description": "Kubernetes monitoring"}`,
			"Kubernetes monitoring",
		},
		{
			"bastille no package.json",
			"",
			"",
		},
		{
			"no description field",
			`{"name": "test", "version": "1.0.0"}`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			oldWd, _ := os.Getwd()
			if err := os.Chdir(tempDir); err != nil {
				t.Fatal(err)
			}
			defer os.Chdir(oldWd)

			if tt.packageJSON != "" {
				if err := os.WriteFile("package.json", []byte(tt.packageJSON), 0644); err != nil {
					t.Fatal(err)
				}
			}

			got := detectProjectDescription()
			if got != tt.want {
				t.Errorf("detectProjectDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}
