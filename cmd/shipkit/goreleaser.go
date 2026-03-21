package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"text/template"
)

type GoReleaserConfig struct {
	ProjectName  string
	BinaryName   string
	MainPath     string
	RepoOwner    string
	RepoName     string
	Description  string
	License      string
	DockerImage  string
	HasNodeJS    bool
	HasChangelog bool
	HasDocker    bool
	DockerFile   string
}

func runGoReleaser(args []string) error {
	fs := newFlagSet("goreleaser")

	projectName := fs.String("project", "", "Project name (required)")
	binaryName := fs.String("binary", "", "Binary name (defaults to project name)")
	mainPath := fs.String("main", "", "Main package path (defaults to ./cmd/{project})")
	repoOwner := fs.String("owner", "", "Repository owner (required)")
	repoName := fs.String("repo", "", "Repository name (defaults to project name)")
	description := fs.String("description", "", "Project description (required)")
	license := fs.String("license", DefaultLicense, "License type")
	dockerImage := fs.String("docker-image", "", "Docker image name (e.g. owner/project, auto-detected if not provided)")
	outputFile := fs.String("output", FileGoReleaser, "Output file path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *repoOwner == "" {
		return fmt.Errorf("owner is required")
	}

	// Auto-detect project name from go.mod if not provided
	if *projectName == "" {
		*projectName = detectProjectName()
	}
	if *projectName == "" {
		return fmt.Errorf("project is required (could not auto-detect from go.mod)")
	}

	// Auto-detect description from package.json if not provided
	if *description == "" {
		if det := detectProjectDescription(); det != "" {
			*description = det
		} else {
			*description = "Application built with Go"
		}
	}

	// Set defaults
	if *binaryName == "" {
		*binaryName = *projectName
	}
	if *mainPath == "" {
		*mainPath = fmt.Sprintf("./cmd/%s", *projectName)
	}
	if *repoName == "" {
		*repoName = *projectName
	}
	if *dockerImage == "" {
		*dockerImage = fmt.Sprintf("%s/%s", *repoOwner, *projectName)
	}

	// Log all inputs
	logInputs(map[string]string{
		"project":      *projectName,
		"binary":       *binaryName,
		"main":         *mainPath,
		"owner":        *repoOwner,
		"repo":         *repoName,
		"description":  *description,
		"license":      *license,
		"docker-image": *dockerImage,
		"output":       *outputFile,
	})

	// Detect project types and features
	fmt.Println("::group::Detect")
	detected := detectProjectTypes()

	// Check for Node.js
	hasNodeJS := hasProjectType(detected, "Node")
	if !hasNodeJS {
		fmt.Fprintln(os.Stderr, "  No package.json found - skipping Node.js build steps")
	}

	// Check for changelog opt-in
	hasChangelog := fileExists(".goreleaser-changelog")
	if hasChangelog {
		fmt.Fprintln(os.Stderr, "  📝 Detected .goreleaser-changelog - will auto-generate changelog from commits")
	} else {
		fmt.Fprintln(os.Stderr, "  📝 No .goreleaser-changelog - changelog auto-generation disabled (create .goreleaser-changelog to opt-in)")
	}

	// Check for Docker/Containerfile
	hasDocker, dockerFile := detectDockerFiles("goreleaser")
	if hasDocker {
		fmt.Fprintf(os.Stderr, "  🐳 Detected %s - will publish Docker image\n", dockerFile)
	} else {
		fmt.Fprintln(os.Stderr, "  No Containerfile.goreleaser or Dockerfile.goreleaser found - skipping Docker publishing")
	}

	config := GoReleaserConfig{
		ProjectName:  *projectName,
		BinaryName:   *binaryName,
		MainPath:     *mainPath,
		RepoOwner:    *repoOwner,
		RepoName:     *repoName,
		Description:  *description,
		License:      *license,
		DockerImage:  *dockerImage,
		HasNodeJS:    hasNodeJS,
		HasChangelog: hasChangelog,
		HasDocker:    hasDocker,
		DockerFile:   dockerFile,
	}

	fmt.Println("::endgroup::")
	fmt.Println("::group::Generate config")
	if err := generateGoReleaserConfig(config, *outputFile); err != nil {
		fmt.Println("::endgroup::")
		return err
	}
	content, err := os.ReadFile(*outputFile)
	if err == nil {
		fmt.Fprintln(os.Stderr, string(content))
	}
	fmt.Println("::endgroup::")
	return nil
}

//go:embed templates/*.tmpl
var embeddedTemplates embed.FS

func generateGoReleaserConfig(config GoReleaserConfig, outputPath string) error {
	tmpl, err := loadAndParseGoReleaserTemplates()
	if err != nil {
		return err
	}

	// Create output directory if needed
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()

	if err := tmpl.ExecuteTemplate(f, "goreleaser.yml.tmpl", config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Fprintf(os.Stderr, "  📦 Generated GoReleaser config: %s\n", outputPath)
	return nil
}

// loadAndParseGoReleaserTemplates loads and parses all goreleaser templates with a switchable path strategy:
//  1. First tries to load from external files at project root: templates/*.tmpl
//     This allows users to customize templates without recompiling
//  2. Falls back to embedded templates (compiled into the binary)
//     This ensures the binary is self-contained and works anywhere
func loadAndParseGoReleaserTemplates() (*template.Template, error) {
	// Try loading from external files first
	if tmpl, err := loadExternalTemplates(); err == nil {
		return tmpl, nil
	}

	// Fall back to embedded templates
	return loadEmbeddedTemplates()
}

func loadExternalTemplates() (*template.Template, error) {
	templateDirs := []string{
		"templates",       // Running from project root
		"../../templates", // Running tests from cmd/shipkit
	}

	// Also try relative to the source file location
	if _, file, _, ok := runtime.Caller(0); ok {
		sourceDir := filepath.Dir(file)
		projectRoot := filepath.Dir(filepath.Dir(sourceDir))
		templateDirs = append(templateDirs, filepath.Join(projectRoot, "templates"))
	}

	for _, dir := range templateDirs {
		mainTemplatePath := filepath.Join(dir, "goreleaser.yml.tmpl")
		if _, err := os.Stat(mainTemplatePath); err != nil {
			continue
		}

		// Found the directory, load all templates
		tmpl := template.New("goreleaser.yml.tmpl")

		// Parse all .tmpl files in the directory
		pattern := filepath.Join(dir, "*.tmpl")
		files, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				return nil, fmt.Errorf("failed to read template %s: %w", file, err)
			}

			name := filepath.Base(file)
			if _, err := tmpl.New(name).Parse(string(content)); err != nil {
				return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
			}
		}

		return tmpl, nil
	}

	return nil, fmt.Errorf("external templates not found")
}

func loadEmbeddedTemplates() (*template.Template, error) {
	tmpl := template.New("goreleaser.yml.tmpl")

	entries, err := embeddedTemplates.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded templates: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		content, err := embeddedTemplates.ReadFile(filepath.Join("templates", name))
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded template %s: %w", name, err)
		}

		if _, err := tmpl.New(name).Parse(string(content)); err != nil {
			return nil, fmt.Errorf("failed to parse embedded template %s: %w", name, err)
		}
	}

	return tmpl, nil
}
