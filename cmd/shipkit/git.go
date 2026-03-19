package main

import (
	"fmt"
	"os"
	"os/exec"
)

func runGitConfig(args []string) error {
	fs := newFlagSet("git-config")
	userName := fs.String("user-name", "github-actions[bot]", "Git user name")
	userEmail := fs.String("user-email", "github-actions[bot]@users.noreply.github.com", "Git user email")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := exec.Command("git", "config", "user.name", *userName).Run(); err != nil {
		return fmt.Errorf("failed to set git user.name: %w", err)
	}

	if err := exec.Command("git", "config", "user.email", *userEmail).Run(); err != nil {
		return fmt.Errorf("failed to set git user.email: %w", err)
	}

	fmt.Printf("✓ Configured git user: %s <%s>\n", *userName, *userEmail)
	return nil
}

func runGitTag(args []string) error {
	fs := newFlagSet("git-tag")
	tag := fs.String("tag", "", "Tag name (required)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *tag == "" {
		return fmt.Errorf("tag is required")
	}

	return createGitTag(*tag)
}

func createGitTag(tag string) error {
	userName := "github-actions[bot]"
	userEmail := "github-actions[bot]@users.noreply.github.com"

	// Config user
	if err := exec.Command("git", "config", "user.name", userName).Run(); err != nil {
		return fmt.Errorf("failed to set git user.name: %w", err)
	}
	if err := exec.Command("git", "config", "user.email", userEmail).Run(); err != nil {
		return fmt.Errorf("failed to set git user.email: %w", err)
	}

	// Create tag
	msg := fmt.Sprintf("Release %s", tag)
	if err := exec.Command("git", "tag", "-a", tag, "-m", msg).Run(); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}
	fmt.Printf("✓ Created tag: %s\n", tag)

	// Push
	if err := exec.Command("git", "push", "origin", tag).Run(); err != nil {
		return fmt.Errorf("failed to push tag: %w", err)
	}
	fmt.Printf("✓ Pushed tag to origin: %s\n", tag)

	return nil
}

func runGitCleanupTag(args []string) error {
	fs := newFlagSet("git-cleanup-tag")
	tag := fs.String("tag", "", "Tag name to delete (required)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *tag == "" {
		return fmt.Errorf("tag is required")
	}

	// Delete remote
	if err := exec.Command("git", "push", "--delete", "origin", *tag).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Warning: Could not delete remote tag %s: %v\n", *tag, err)
	} else {
		fmt.Printf("🧹 Deleted remote tag: %s\n", *tag)
	}

	// Delete local
	if err := exec.Command("git", "tag", "-d", *tag).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Warning: Could not delete local tag %s: %v\n", *tag, err)
	} else {
		fmt.Printf("🧹 Deleted local tag: %s\n", *tag)
	}

	return nil
}

func runCheckDocker(args []string) error {
	fs := newFlagSet("check-docker")
	if err := fs.Parse(args); err != nil {
		return err
	}

	hasGoreleaserDocker := fileExists(FileGoreleaserContainerfile) || fileExists(FileGoreleaserDockerfile)

	githubOutput := os.Getenv(EnvGitHubOutput)

	if hasGoreleaserDocker {
		writeOutput(githubOutput, "goreleaser_docker", PublishTrue)
		fmt.Println("GoReleaser will handle Docker builds")
	} else {
		writeOutput(githubOutput, "goreleaser_docker", PublishFalse)
		fmt.Println("Docker workflow will handle Docker builds")
	}

	return nil
}
