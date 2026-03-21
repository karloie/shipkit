package main

import (
	"fmt"
	"os"
)

func configureGitUser(userName, userEmail string) error {
	if err := defaultRunner.Run("git", "config", "user.name", userName); err != nil {
		return fmt.Errorf("failed to set git user.name: %w", err)
	}
	if err := defaultRunner.Run("git", "config", "user.email", userEmail); err != nil {
		return fmt.Errorf("failed to set git user.email: %w", err)
	}
	return nil
}

func runGitConfig(args []string) error {
	fs := newFlagSet("git-config")
	userName := fs.String("user-name", "github-actions[bot]", "Git user name")
	userEmail := fs.String("user-email", "github-actions[bot]@users.noreply.github.com", "Git user email")

	if err := fs.Parse(args); err != nil {
		return err
	}

	logInputs(map[string]string{
		"user-name":  *userName,
		"user-email": *userEmail,
	})

	if err := configureGitUser(*userName, *userEmail); err != nil {
		return err
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

	logInputs(map[string]string{
		"tag": *tag,
	})

	return createGitTag(*tag)
}

func createGitTag(tag string) error {
	userName := "github-actions[bot]"
	userEmail := "github-actions[bot]@users.noreply.github.com"

	// Config user
	if err := configureGitUser(userName, userEmail); err != nil {
		return err
	}

	// Fetch remote tags to check if tag already exists
	if err := defaultRunner.Run("git", "fetch", "--tags"); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: failed to fetch tags: %v\n", err)
	}

	// Check if tag already exists
	git := &GitProviderReal{}
	exists, _ := git.TagExists(tag)
	if exists {
		fmt.Printf("⚠️  Tag %s already exists, using -f to force update\n", tag)
	}

	// Create tag (force if exists)
	msg := fmt.Sprintf("Release %s", tag)
	if err := defaultRunner.Run("git", "tag", "-f", "-a", tag, "-m", msg); err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}
	fmt.Printf("✓ Created tag: %s\n", tag)

	// Push (force to overwrite remote if exists)
	if err := defaultRunner.Run("git", "push", "-f", "origin", tag); err != nil {
		return fmt.Errorf("failed to push tag: %w", err)
	}
	fmt.Printf("✓ Pushed tag to origin: %s\n", tag)

	return nil
}

func runGitTagCleanup(args []string) error {
	fs := newFlagSet("git-tag-cleanup")
	tag := fs.String("tag", "", "Tag name to delete (required)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *tag == "" {
		return fmt.Errorf("tag is required")
	}

	// Delete remote
	if err := defaultRunner.Run("git", "push", "--delete", "origin", *tag); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Warning: Could not delete remote tag %s: %v\n", *tag, err)
	} else {
		fmt.Printf("🧹 Deleted remote tag: %s\n", *tag)
	}

	// Delete local
	if err := defaultRunner.Run("git", "tag", "-d", *tag); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Warning: Could not delete local tag %s: %v\n", *tag, err)
	} else {
		fmt.Printf("🧹 Deleted local tag: %s\n", *tag)
	}

	return nil
}

func runDockerHubStatus(args []string) error {
	fs := newFlagSet("docker-hub-status")
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
