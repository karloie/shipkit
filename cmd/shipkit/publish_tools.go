package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// PlanData contains the subset of plan.json fields needed by publish commands
type PlanData struct {
	Tag         string `json:"tag"`
	Version     string `json:"version"`
	DockerImage string `json:"docker_image"`
}

// loadPlanData loads plan.json if it exists, returns nil if not found
func loadPlanData(path string) (*PlanData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not an error, just no plan file
		}
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan PlanData
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	return &plan, nil
}

// runPublishGoreleaser executes goreleaser to publish releases
func runPublishGoreleaser(args []string) error {
	// Log raw args BEFORE parsing
	logInputs(map[string]string{
		"raw_args": strings.Join(args, " "),
	})
	fs := newFlagSet("publish-goreleaser")
	config := fs.String("config", ".goreleaser.yml", "GoReleaser config")
	planFile := fs.String("plan-file", "plan.json", "Plan file path")
	snapshot := fs.Bool("snapshot", false, "Build snapshot")
	skipPublish := fs.Bool("skip-publish", false, "Skip publishing")
	skipDocker := fs.Bool("skip-docker", false, "Skip Docker")
	clean := fs.Bool("clean", true, "Clean dist/")
	parseFlagsOrExit(fs, args)

	// Log inputs
	logInputs(map[string]string{
		"config":       *config,
		"plan-file":    *planFile,
		"snapshot":     fmt.Sprintf("%v", *snapshot),
		"skip-publish": fmt.Sprintf("%v", *skipPublish),
		"skip-docker":  fmt.Sprintf("%v", *skipDocker),
		"clean":        fmt.Sprintf("%v", *clean),
	})

	_ = *planFile // Reserved for future use

	// Check if goreleaser is installed
	if _, err := exec.LookPath("goreleaser"); err != nil {
		return fmt.Errorf("goreleaser not found in PATH. Install it from https://goreleaser.com/install/")
	}

	// Build goreleaser command
	cmdArgs := []string{"release"}

	if *config != "" && *config != ".goreleaser.yml" {
		cmdArgs = append(cmdArgs, "--config", *config)
	}

	if *snapshot {
		cmdArgs = append(cmdArgs, "--snapshot")
	}

	if *skipPublish {
		cmdArgs = append(cmdArgs, "--skip-publish")
	}

	if *skipDocker {
		cmdArgs = append(cmdArgs, "--skip=docker")
	}

	if *clean {
		cmdArgs = append(cmdArgs, "--clean")
	}

	fmt.Fprintf(os.Stderr, "🚀 Running: goreleaser %s\n", strings.Join(cmdArgs, " "))

	cmd := exec.Command("goreleaser", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("goreleaser failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ GoReleaser publish completed successfully\n")

	// Log outputs
	logOutputs(map[string]string{
		"status": "success",
	})

	return nil
}

// runPublishDocker builds and publishes Docker images
func runPublishDocker(args []string) error {
	// Log raw args BEFORE parsing
	logInputs(map[string]string{
		"raw_args": strings.Join(args, " "),
	})
	fs := newFlagSet("publish-docker")
	planFile := fs.String("plan-file", "plan.json", "Plan file path")
	image := fs.String("image", "", "Image name")
	tag := fs.String("tag", "", "Image tag")
	tagLatest := fs.Bool("tag-latest", false, "Tag as latest")
	platform := fs.String("platform", "linux/amd64,linux/arm64", "Build platforms")
	dockerfile := fs.String("dockerfile", "Dockerfile", "Dockerfile path")
	context := fs.String("context", ".", "Build context")
	push := fs.Bool("push", true, "Push to registry")
	updateReadme := fs.Bool("update-readme", true, "Update Docker Hub README")
	readmePath := fs.String("readme", "README.md", "README path")
	parseFlagsOrExit(fs, args)

	// Log inputs
	logInputs(map[string]string{
		"plan-file":     *planFile,
		"image":         *image,
		"tag":           *tag,
		"tag-latest":    fmt.Sprintf("%v", *tagLatest),
		"platform":      *platform,
		"dockerfile":    *dockerfile,
		"context":       *context,
		"push":          fmt.Sprintf("%v", *push),
		"update-readme": fmt.Sprintf("%v", *updateReadme),
		"readme":        *readmePath,
	})

	plan := loadPlanOrWarn(*planFile)
	if plan != nil {
		if *image == "" && plan.DockerImage != "" {
			*image = plan.DockerImage
		}
		if *tag == "" && plan.Tag != "" {
			*tag = plan.Tag
		}
	}

	// Validate required fields
	if *image == "" {
		return fmt.Errorf("image is required (provide via -image flag or plan.json)")
	}
	if *tag == "" {
		*tag = "latest"
	}

	// Check if docker is installed
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not found in PATH")
	}

	// Login to Docker registry if credentials are available
	username := os.Getenv("DOCKERHUB_USERNAME")
	token := getSecretWithFallbacks("DOCKERHUB_TOKEN", "DOCKERHUB_PASSWORD")

	if username != "" && token != "" {
		fmt.Println("::group::Docker Login")
		if err := dockerLogin(username, token); err != nil {
			fmt.Println("::endgroup::")
			return fmt.Errorf("docker login failed: %w", err)
		}
		fmt.Println("::endgroup::")
	}

	// Build the image
	fmt.Println("::group::Docker Build")
	fullTag := *image + ":" + *tag

	buildArgs := []string{"buildx", "build"}
	buildArgs = append(buildArgs, "--platform", *platform)
	buildArgs = append(buildArgs, "-f", *dockerfile)
	buildArgs = append(buildArgs, "-t", fullTag)

	if *tagLatest {
		latestTag := *image + ":latest"
		buildArgs = append(buildArgs, "-t", latestTag)
	}

	if *push {
		buildArgs = append(buildArgs, "--push")
	} else {
		buildArgs = append(buildArgs, "--load")
	}

	buildArgs = append(buildArgs, *context)

	fmt.Fprintf(os.Stderr, "🐳 Building: docker %s\n", strings.Join(buildArgs, " "))

	cmd := exec.Command("docker", buildArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		fmt.Println("::endgroup::")
		return fmt.Errorf("docker build failed: %w", err)
	}
	fmt.Println("::endgroup::")

	// Update Docker Hub README if requested and credentials available
	if *push && *updateReadme && username != "" && token != "" {
		fmt.Println("::group::Update Docker Hub README")
		if err := runDockerHubReadme([]string{
			"-repo", *image,
			"-username", username,
			"-password", token,
			"-readme", *readmePath,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Failed to update Docker Hub README: %v\n", err)
		}
		fmt.Println("::endgroup::")
	}

	fmt.Fprintf(os.Stderr, "✅ Docker publish completed successfully\n")

	// Log outputs
	logOutputs(map[string]string{
		"status": "success",
		"image":  *image,
		"tag":    *tag,
		"pushed": fmt.Sprintf("%v", *push),
	})

	return nil
}
