package main

import (
	"fmt"
	"os"
	"path/filepath"
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

func runGoReleaserGenerate(args []string) error {
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

	if *projectName == "" || *repoOwner == "" || *description == "" {
		return fmt.Errorf("project, owner, and description are required")
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

	// Detect project types and features
	detected := detectProjectTypes()

	// Check for Node.js
	hasNodeJS := hasProjectType(detected, "Node.js")
	if !hasNodeJS {
		fmt.Fprintln(os.Stderr, "  No package.json found - skipping Node.js build steps")
	}

	// Check for changelog
	hasChangelog := fileExists(FileChangelog)
	if hasChangelog {
		fmt.Fprintln(os.Stderr, "📝 Detected CHANGELOG.md - will use existing changelog")
	} else {
		fmt.Fprintln(os.Stderr, "  No CHANGELOG.md found - will use GitHub auto-generated changelog")
	}

	// Check for Docker/Containerfile
	hasDocker, dockerFile := detectDockerFiles("goreleaser")
	if hasDocker {
		fmt.Fprintf(os.Stderr, "🐳 Detected %s - will publish Docker image\n", dockerFile)
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

	return generateGoReleaserConfig(config, *outputFile)
}

func generateGoReleaserConfig(config GoReleaserConfig, outputPath string) error {
	tmpl, err := template.New("goreleaser").Parse(defaultGoReleaserTemplate())
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
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

	if err := tmpl.Execute(f, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	fmt.Fprintf(os.Stderr, "📦 Generated GoReleaser config: %s\n", outputPath)
	return nil
}

func defaultGoReleaserTemplate() string {
	return `version: 2

before:
  hooks:
    - go mod tidy{{if .HasNodeJS}}
    - npm ci
    - npm run build{{end}}

builds:
  - id: {{.ProjectName}}
    main: {{.MainPath}}
    binary: {{.BinaryName}}
    env:
      - CGO_ENABLED=0
    flags:
      - -tags=release
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.buildCommit={{ "{{" }}.Commit{{ "}}" }}
      - -X main.buildDate={{ "{{" }}.Date{{ "}}" }}
      - -X main.buildVersion={{ "{{" }}.Version{{ "}}" }}

universal_binaries:
  - ids:
      - {{.ProjectName}}
    replace: true

archives:
  - name_template: "{{ "{{" }} .ProjectName {{ "}}" }}_{{ "{{" }} .Version {{ "}}" }}_{{ "{{" }} .Os {{ "}}" }}_{{ "{{" }} .Arch {{ "}}" }}"
    files:
      - LICENSE*
      - README*
      - CHANGELOG*
      - completions/*
      - manpages/*

nfpms:
  - id: packages
    package_name: {{.ProjectName}}
    homepage: https://github.com/{{.RepoOwner}}/{{.RepoName}}
    maintainer: {{.RepoOwner}}
    description: {{.Description}}
    license: {{.License}}

brews:
  - repository:
      owner: {{.RepoOwner}}
      name: homebrew-tap
      token: "{{ "{{" }} .Env.HOMEBREW_TAP_GITHUB_TOKEN {{ "}}" }}"
    homepage: https://github.com/{{.RepoOwner}}/{{.RepoName}}
    description: {{.Description}}
    license: {{.License}}
{{if .HasDocker}}
dockers:
  - image_templates:
      - "{{.DockerImage}}:{{ "{{" }} .Tag {{ "}}" }}"
      - "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}"
      - "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}.{{ "{{" }} .Minor {{ "}}" }}"
      - "{{.DockerImage}}:latest"
    dockerfile: {{.DockerFile}}
    use: buildx
    goos: linux
    goarch: amd64
    build_flag_templates:
      - "--platform=linux/amd64"
      - "--label=org.opencontainers.image.title={{.ProjectName}}"
      - "--label=org.opencontainers.image.description={{.Description}}"
      - "--label=org.opencontainers.image.url=https://github.com/{{.RepoOwner}}/{{.RepoName}}"
      - "--label=org.opencontainers.image.source=https://github.com/{{.RepoOwner}}/{{.RepoName}}"
      - "--label=org.opencontainers.image.version={{ "{{" }} .Version {{ "}}" }}"
      - "--label=org.opencontainers.image.created={{ "{{" }} .Date {{ "}}" }}"
      - "--label=org.opencontainers.image.revision={{ "{{" }} .FullCommit {{ "}}" }}"
      - "--label=org.opencontainers.image.licenses={{.License}}"
  - image_templates:
      - "{{.DockerImage}}:{{ "{{" }} .Tag {{ "}}" }}-arm64"
      - "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}-arm64"
      - "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}.{{ "{{" }} .Minor {{ "}}" }}-arm64"
      - "{{.DockerImage}}:latest-arm64"
    dockerfile: {{.DockerFile}}
    use: buildx
    goos: linux
    goarch: arm64
    build_flag_templates:
      - "--platform=linux/arm64"
      - "--label=org.opencontainers.image.title={{.ProjectName}}"
      - "--label=org.opencontainers.image.description={{.Description}}"
      - "--label=org.opencontainers.image.url=https://github.com/{{.RepoOwner}}/{{.RepoName}}"
      - "--label=org.opencontainers.image.source=https://github.com/{{.RepoOwner}}/{{.RepoName}}"
      - "--label=org.opencontainers.image.version={{ "{{" }} .Version {{ "}}" }}"
      - "--label=org.opencontainers.image.created={{ "{{" }} .Date {{ "}}" }}"
      - "--label=org.opencontainers.image.revision={{ "{{" }} .FullCommit {{ "}}" }}"
      - "--label=org.opencontainers.image.licenses={{.License}}"

docker_manifests:
  - name_template: "{{.DockerImage}}:{{ "{{" }} .Tag {{ "}}" }}"
    image_templates:
      - "{{.DockerImage}}:{{ "{{" }} .Tag {{ "}}" }}"
      - "{{.DockerImage}}:{{ "{{" }} .Tag {{ "}}" }}-arm64"
  - name_template: "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}"
    image_templates:
      - "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}"
      - "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}-arm64"
  - name_template: "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}.{{ "{{" }} .Minor {{ "}}" }}"
    image_templates:
      - "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}.{{ "{{" }} .Minor {{ "}}" }}"
      - "{{.DockerImage}}:{{ "{{" }} .Major {{ "}}" }}.{{ "{{" }} .Minor {{ "}}" }}-arm64"
  - name_template: "{{.DockerImage}}:latest"
    image_templates:
      - "{{.DockerImage}}:latest"
      - "{{.DockerImage}}:latest-arm64"
{{end}}
source: {}

checksum:
  name_template: 'checksums.txt'

sboms:
  - artifacts: archive
{{if .HasChangelog}}
changelog:
  skip: true
{{else}}
changelog:
  use: github
  sort: asc
  filters:
    exclude:
      - '^docs'
      - '^test'
      - '^chore'
      - Merge pull request
      - Merge branch
{{end}}
`
}
