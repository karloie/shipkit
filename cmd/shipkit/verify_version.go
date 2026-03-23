package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func runVerifyVersion(args []string) error {
	fs := newFlagSet("verify-version")
	fileType := fs.String("type", "", "File type")
	version := fs.String("version", "", "Version")
	tag := fs.String("tag", "", "Tag")
	fix := fs.Bool("fix", false, "Fix version")
	parseFlagsOrExit(fs, args)

	if *fileType == "" {
		return fmt.Errorf("-type is required (npm or maven)")
	}

	// Compute expected version: use version if provided, otherwise strip 'v' from tag
	var expectedVersion string
	if *version != "" {
		expectedVersion = *version
	} else if *tag != "" {
		expectedVersion = strings.TrimPrefix(*tag, "v")
	} else {
		return fmt.Errorf("either -version or -tag must be provided")
	}

	// Log all inputs
	logInputs(map[string]string{
		"type":    *fileType,
		"version": expectedVersion,
		"tag":     *tag,
		"fix":     fmt.Sprintf("%v", *fix),
	})

	var currentVersion string
	var err error

	switch *fileType {
	case "npm":
		currentVersion, err = getNpmVersion()
		if err != nil {
			return fmt.Errorf("failed to read package.json version: %w", err)
		}
	case "maven":
		currentVersion, err = getMavenVersion()
		if err != nil {
			return fmt.Errorf("failed to read pom.xml version: %w", err)
		}
	default:
		return fmt.Errorf("invalid type: %s (must be npm or maven)", *fileType)
	}

	if currentVersion != expectedVersion {
		if *fix {
			fmt.Printf("⚠️  Version mismatch: current=%s expected=%s - fixing...\n", currentVersion, expectedVersion)
			if err := setVersion(*fileType, expectedVersion); err != nil {
				return fmt.Errorf("failed to set version: %w", err)
			}
			fmt.Printf("✓ Updated version to %s\n", expectedVersion)
		} else {
			return fmt.Errorf("version mismatch: current=%s expected=%s", currentVersion, expectedVersion)
		}
	} else {
		fmt.Printf("✓ Version verified: %s\n", currentVersion)
	}

	// Write computed version to GITHUB_OUTPUT for workflow use
	if githubOutput := os.Getenv("GITHUB_OUTPUT"); githubOutput != "" {
		writeOutput(githubOutput, "version", expectedVersion)
	}

	return nil
}

func getNpmVersion() (string, error) {
	packagePath := filepath.Join(".", "package.json")
	data, err := os.ReadFile(packagePath)
	if err != nil {
		return "", err
	}

	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", err
	}

	return pkg.Version, nil
}

func getMavenVersion() (string, error) {
	pomPath := filepath.Join(".", "pom.xml")
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return "", err
	}

	var pom struct {
		Version string `xml:"version"`
	}
	if err := xml.Unmarshal(data, &pom); err != nil {
		return "", err
	}

	return pom.Version, nil
}

func setVersion(fileType, version string) error {
	switch fileType {
	case "npm":
		cmd := exec.Command("npm", "version", version, "--no-git-tag-version", "--allow-same-version")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "maven":
		cmd := exec.Command("mvn", "versions:set", fmt.Sprintf("-DnewVersion=%s", version), "-DgenerateBackupPoms=false")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported file type: %s", fileType)
	}
}
